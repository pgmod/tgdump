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

	doBackup(cfg)
	// Создаем дампы для всех баз

	scheduler.ScheduleDailyAt("08:00", func() {
		doBackup(cfg)
	})

	select {} // блокируем завершение программы
}

func doBackup(cfg *config.Config) {
	nowTs := time.Now().Format("2006-01-02_15-04-05")
	dumpDir := filepath.Join(cfg.DumpDir, nowTs)
	os.MkdirAll(dumpDir, 0755)
	for _, db := range cfg.Databases {
		err := backup.DumpDatabaseEx(db, filepath.Join(dumpDir, db.DBName+".sql"))
		if err != nil {
			log.Fatal(err)
		}
	}
	for _, file := range cfg.Files {
		fmt.Println("copy file", filepath.Join("./files", file), filepath.Join(dumpDir, filepath.Base(file)))
		err := backup.CopyFile(filepath.Join("./files", file), filepath.Join(dumpDir, filepath.Base(file)))
		if err != nil {
			log.Fatal(err)
		}
	}
	for _, dir := range cfg.Directories {
		fmt.Println("copy dir", filepath.Join("./files", dir), filepath.Join(dumpDir, filepath.Base(dir)))
		err := backup.CopyDir(filepath.Join("./files", dir), filepath.Join(dumpDir, filepath.Base(dir)))
		if err != nil {
			log.Fatal(err)
		}
	}
	err := telegram.SendFolder(cfg.Telegram.Token, cfg.Telegram.ChatID, dumpDir)
	if err != nil {
		log.Fatal(err)
	}
}
