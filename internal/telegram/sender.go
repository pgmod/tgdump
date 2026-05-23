package telegram

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tgdump/internal/archive"
)

const maxTelegramMessageLen = 4096

// SendMessage отправляет текстовое сообщение в чат Telegram.
func SendMessage(token, chatID, text string) error {
	if len(text) > maxTelegramMessageLen {
		text = text[:maxTelegramMessageLen-3] + "..."
	}

	form := url.Values{
		"chat_id": {chatID},
		"text":    {text},
	}

	req, err := http.NewRequest(http.MethodPost,
		"https://api.telegram.org/bot"+token+"/sendMessage",
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("не удалось создать запрос: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := telegramHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка ответа Telegram: %s", string(respBody))
	}
	return nil
}

func telegramHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

// SendFolder архивирует каталог и отправляет zip в Telegram.
// keepZip: сохранить zip на диске после отправки.
func SendFolder(token, chatID, folderPath string, keepZip bool) error {
	log.Printf("создание архива для отправки: %s", folderPath)
	zipPath, err := archive.ZipDirectory(folderPath)
	if err != nil {
		return err
	}
	if !keepZip {
		defer func() {
			if err := os.Remove(zipPath); err != nil {
				log.Printf("не удалось удалить временный архив: %v", err)
			}
		}()
	}

	zipInfo, err := os.Stat(zipPath)
	if err != nil {
		return fmt.Errorf("ошибка получения информации о файле: %w", err)
	}
	log.Printf("размер архива для отправки: %d bytes", zipInfo.Size())

	log.Printf("отправка архива %s", zipPath)
	if err := SendFile(token, chatID, zipPath); err != nil {
		return fmt.Errorf("ошибка отправки архива: %w", err)
	}
	if keepZip {
		log.Printf("архив сохранён: %s", zipPath)
	}
	return nil
}

// SendFile отправляет файл в чат Telegram.
func SendFile(token, chatID, filePath string) error {
	log.Printf("отправка файла %s", filePath)

	chat, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("ошибка конвертации chatID: %w", err)
	}

	err = SendFileWithProgress(token, chat, filePath)
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

	resp, err := telegramHTTPClient().Do(req)
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
