package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ZipDirectory(dir string) (string, error) {
	zipPath := dir + ".zip"

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("ошибка создания архива: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("ошибка обхода %s: %w", path, err)
		}
		if info.IsDir() {
			return nil
		}
		return addFileToZip(zipWriter, dir, path)
	})
	if err != nil {
		zipWriter.Close()
		return "", err
	}
	if err := zipWriter.Close(); err != nil {
		return "", fmt.Errorf("ошибка закрытия архива: %w", err)
	}
	return zipPath, nil
}

func addFileToZip(zipWriter *zip.Writer, baseDir, path string) error {
	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return fmt.Errorf("ошибка вычисления относительного пути: %w", err)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла %s: %w", path, err)
	}
	defer file.Close()

	zipEntry, err := zipWriter.Create(relPath)
	if err != nil {
		return fmt.Errorf("ошибка создания zip-записи для %s: %w", path, err)
	}

	if _, err := io.Copy(zipEntry, file); err != nil {
		return fmt.Errorf("ошибка записи файла %s в архив: %w", path, err)
	}
	return nil
}
