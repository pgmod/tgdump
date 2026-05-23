package main

import (
	"log"

	"tgdump/internal/backup"
	"tgdump/internal/config"
	"tgdump/internal/scheduler"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}
	cfg.Print()

	if err := backup.Run(cfg); err != nil {
		log.Fatal(err)
	}

	scheduler.ScheduleDailyAt(cfg.Schedule, func() {
		if err := backup.Run(cfg); err != nil {
			log.Printf("ошибка при выполнении резервного копирования: %v", err)
		} else {
			log.Printf("резервное копирование выполнено успешно")
		}
	})

	select {}
}
