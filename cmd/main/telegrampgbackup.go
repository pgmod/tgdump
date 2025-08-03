package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"tgdump/internal/backup"
	"tgdump/internal/config"
	"tgdump/internal/scheduler"
	"tgdump/internal/telegram"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}
	cfg.Print()

	err = doBackup(cfg)
	if err != nil {
		log.Fatal(err)
	}
	// Создаем дампы для всех баз

	scheduler.ScheduleDailyAt("08:00", func() {
		err := doBackup(cfg)
		if err != nil {
			log.Printf("ошибка при выполнении резервного копирования: %v", err)
		} else {
			log.Printf("резервное копирование выполнено успешно")
		}
	})

	select {} // блокируем завершение программы
}

func doBackup(cfg *config.Config) error {
	nowTs := time.Now().Format("2006-01-02_15-04-05")
	dumpDir := filepath.Join(cfg.DumpDir, nowTs)
	os.MkdirAll(dumpDir, 0755)
	for _, db := range cfg.Databases {
		err := backup.DumpDatabaseEx(db, filepath.Join(dumpDir, db.DBName+".sql"))
		if err != nil {
			return err
		}
	}
	for _, file := range cfg.Files {
		fmt.Println("copy file", filepath.Join("./files", file), filepath.Join(dumpDir, filepath.Base(file)))
		err := backup.CopyFile(filepath.Join("./files", file), filepath.Join(dumpDir, filepath.Base(file)))
		if err != nil {
			return err
		}
		fmt.Println("done copy file", filepath.Join("./files", file), filepath.Join(dumpDir, filepath.Base(file)))
	}
	for _, dir := range cfg.Directories {
		fmt.Println("copy dir", filepath.Join("./files", dir), filepath.Join(dumpDir, filepath.Base(dir)))
		err := backup.CopyDir(filepath.Join("./files", dir), filepath.Join(dumpDir, filepath.Base(dir)))
		if err != nil {
			return err
		}
		fmt.Println("done copy dir", filepath.Join("./files", dir), filepath.Join(dumpDir, filepath.Base(dir)))
	}
	defer func() {
		err := os.RemoveAll(dumpDir)
		if err != nil {
			fmt.Printf("не удалось удалить директорию: %v", err)
		}
	}()
	err := telegram.SendFolder(cfg.Telegram.Token, cfg.Telegram.ChatID, dumpDir)
	if err != nil {
		return err
	}
	return nil
}
