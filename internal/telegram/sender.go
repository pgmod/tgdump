package telegram

import (
	"archive/zip"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
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

	chat, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("ошибка конвертации chatID: %w", err)
	}

	err = SendFileWithProgress(token, chat, filepath)
	if err != nil {
		return fmt.Errorf("ошибка отправки файла: %w", err)
	}

	return nil
}

type ProgressReader struct {
	io.Reader
	Total      int64
	ReadSoFar  int64
	LastUpdate time.Time
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.ReadSoFar += int64(n)

	// Показываем прогресс не чаще раза в 300 мс
	if time.Since(pr.LastUpdate) > 300*time.Millisecond || pr.ReadSoFar == pr.Total {
		percent := float64(pr.ReadSoFar) / float64(pr.Total) * 100
		log.Printf("Прогресс загрузки: %.2f%%", percent)
		pr.LastUpdate = time.Now()
	}

	return n, err
}

func SendFileWithProgress(token string, chatID int64, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("не удалось получить информацию о файле: %w", err)
	}

	pr := &ProgressReader{
		Reader: file,
		Total:  fileInfo.Size(),
	}

	// Создаём io.Pipe для записи и чтения
	bodyReader, bodyWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(bodyWriter)

	// Пишем тело в фоне
	go func() {
		defer bodyWriter.Close()
		defer multipartWriter.Close()

		// Добавляем поле chat_id
		_ = multipartWriter.WriteField("chat_id", strconv.FormatInt(chatID, 10))

		// Добавляем файл
		part, err := multipartWriter.CreateFormFile("document", filepath.Base(filePath))
		if err != nil {
			bodyWriter.CloseWithError(err)
			return
		}

		_, err = io.Copy(part, pr) // Здесь будет вызываться ProgressReader
		if err != nil {
			bodyWriter.CloseWithError(err)
			return
		}
	}()

	req, err := http.NewRequest("POST", "https://api.telegram.org/bot"+token+"/sendDocument", bodyReader)
	if err != nil {
		return fmt.Errorf("не удалось создать запрос: %w", err)
	}
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // осторожно: только если точно нужно
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("ошибка ответа Telegram: %s", string(respBody))
	}

	log.Println("Файл успешно отправлен.")
	return nil
}
