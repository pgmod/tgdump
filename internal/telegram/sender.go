package telegram

import (
	"archive/zip"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SendFolder(token, chatID, folderPath string) error {
	zipPath := folderPath + ".zip"

	// Создание zip архива
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("ошибка создания архива: %w", err)
	}
	defer zipFile.Close()
	defer func() {
		err = os.Remove(zipPath)
		if err != nil {
			fmt.Printf("не удалось удалить файл: %v", err)
		}
	}()

	log.Printf("создание архива %s", zipPath)
	zipWriter := zip.NewWriter(zipFile)

	// Рекурсивное добавление файлов и папок
	err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("ошибка обхода %s: %w", path, err)
		}
		if info.IsDir() {
			return nil // пропускаем директории, сами они не нужны в zip
		}

		// Открытие файла
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("ошибка открытия файла %s: %w", path, err)
		}
		defer file.Close()

		// Относительный путь в архиве
		relPath, err := filepath.Rel(folderPath, path)
		if err != nil {
			return fmt.Errorf("ошибка вычисления относительного пути: %w", err)
		}

		zipEntry, err := zipWriter.Create(relPath)
		if err != nil {
			return fmt.Errorf("ошибка создания zip-записи для %s: %w", path, err)
		}

		_, err = io.Copy(zipEntry, file)
		if err != nil {
			return fmt.Errorf("ошибка записи файла %s в архив: %w", path, err)
		}

		return nil
	})

	if err != nil {
		zipWriter.Close()
		return err
	}

	err = zipWriter.Close()
	if err != nil {
		return fmt.Errorf("ошибка закрытия архива: %w", err)
	}

	// подсчет размера архива
	zipInfo, err := os.Stat(zipPath)
	if err != nil {
		return fmt.Errorf("ошибка получения информации о файле: %w", err)
	}
	zipSize := zipInfo.Size()
	log.Printf("размер архива: %d bytes", zipSize)

	log.Printf("отправка архива %s", zipPath)
	// Отправка архива
	err = SendFile(token, chatID, zipPath)
	if err != nil {
		return fmt.Errorf("ошибка отправки архива: %w", err)
	}

	// Удаление архива

	return nil
}

// SendFile отправляет файл в чат Telegram
func SendFile(token, chatID, filepath string) error {
	log.Printf("отправка файла %s", filepath)
	bot, err := tgbotapi.NewBotAPIWithClient(token, tgbotapi.APIEndpoint, &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("ошибка создания Telegram-бота: %w", err)
	}

	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer file.Close()

	chat, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("ошибка конвертации chatID: %w", err)
	}

	doc := tgbotapi.NewDocument(chat, tgbotapi.FileReader{
		Name:   filepath,
		Reader: file,
		// Size:   -1, // Telegram сам определит
	})

	_, err = bot.Send(doc)
	if err != nil {
		return fmt.Errorf("ошибка отправки файла: %w", err)
	}

	return nil
}
