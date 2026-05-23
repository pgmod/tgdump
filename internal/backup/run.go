package backup

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"tgdump/internal/archive"
	"tgdump/internal/config"
	"tgdump/internal/telegram"
)

func Run(cfg *config.Config) error {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	archiveDir := filepath.Join(cfg.DumpDir, timestamp)
	sendDir := filepath.Join(cfg.DumpDir, timestamp+"_telegram")

	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return fmt.Errorf("не удалось создать каталог дампа: %w", err)
	}
	if err := os.MkdirAll(sendDir, 0o755); err != nil {
		return fmt.Errorf("не удалось создать каталог для отправки: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(archiveDir)
		_ = os.RemoveAll(sendDir)
	}()

	report := Report{Timestamp: timestamp}

	for _, db := range cfg.Databases {
		outFile := filepath.Join(archiveDir, db.DBName+".sql")
		stats, err := DumpDatabaseEx(db, outFile)
		if err != nil {
			return err
		}
		report.Databases = append(report.Databases, DatabaseReport{
			Name:     db.DBName,
			Delivery: db.Delivery,
			Tables:   stats,
		})
		if db.Delivery.ShouldSend() {
			if err := CopyFile(outFile, filepath.Join(sendDir, db.DBName+".sql")); err != nil {
				return fmt.Errorf("копирование дампа для отправки %s: %w", db.DBName, err)
			}
		}
	}

	dirReports, fileReports, err := copyAssets(cfg.FilesDir, cfg.Files, cfg.Directories, archiveDir, sendDir)
	if err != nil {
		return err
	}
	report.Directories = dirReports
	report.Files = fileReports

	zipPath, err := archive.ZipDirectory(archiveDir)
	if err != nil {
		return fmt.Errorf("создание архива: %w", err)
	}
	log.Printf("архив сохранён: %s", zipPath)

	if err := telegram.SendMessage(cfg.Telegram.Token, cfg.Telegram.ChatID, report.Format()); err != nil {
		return fmt.Errorf("отправка отчёта: %w", err)
	}

	hasSend, err := dirHasFiles(sendDir)
	if err != nil {
		return err
	}
	if !hasSend {
		log.Printf("нет элементов с delivery=send, архив в Telegram не отправляется")
		return nil
	}

	return telegram.SendFolder(cfg.Telegram.Token, cfg.Telegram.ChatID, sendDir, false)
}

func dirHasFiles(dir string) (bool, error) {
	var found bool
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found, err
}

func copyAssets(filesDir string, files, dirs config.AssetList, archiveDir, sendDir string) ([]DirectoryReport, []FileReport, error) {
	var dirReports []DirectoryReport
	var fileReports []FileReport

	for _, entry := range files {
		src := filepath.Join(filesDir, entry.Path)
		name := filepath.Base(entry.Path)
		archiveDst := filepath.Join(archiveDir, name)
		log.Printf("копирование файла %s -> %s", src, archiveDst)
		if err := CopyFile(src, archiveDst); err != nil {
			return nil, nil, fmt.Errorf("копирование файла %s: %w", src, err)
		}
		fileReports = append(fileReports, FileReport{Name: name, Delivery: entry.Delivery})
		if entry.Delivery.ShouldSend() {
			if err := CopyFile(src, filepath.Join(sendDir, name)); err != nil {
				return nil, nil, fmt.Errorf("копирование файла для отправки %s: %w", src, err)
			}
		}
	}

	for _, entry := range dirs {
		src := filepath.Join(filesDir, entry.Path)
		name := filepath.Base(entry.Path)
		archiveDst := filepath.Join(archiveDir, name)

		stat, err := collectDirectoryStats(src, name)
		if err != nil {
			return nil, nil, err
		}
		stat.Delivery = entry.Delivery
		dirReports = append(dirReports, stat)

		log.Printf("копирование каталога %s -> %s", src, archiveDst)
		if err := CopyDir(src, archiveDst); err != nil {
			return nil, nil, fmt.Errorf("копирование каталога %s: %w", src, err)
		}
		if entry.Delivery.ShouldSend() {
			if err := CopyDir(src, filepath.Join(sendDir, name)); err != nil {
				return nil, nil, fmt.Errorf("копирование каталога для отправки %s: %w", src, err)
			}
		}
	}
	return dirReports, fileReports, nil
}
