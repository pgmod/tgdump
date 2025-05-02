package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Копирование файла с сохранением прав доступа
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func CopyDir(from, to string) error {
	fmt.Println("copy dir", from, to)
	from, err := filepath.Abs(from)
	if err != nil {
		return err
	}
	to, err = filepath.Abs(to)
	if err != nil {
		return err
	}

	// Создание целевой директории
	err = os.MkdirAll(to, 0755)
	if err != nil {
		return err
	}

	// Обход исходной директории
	return filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(to, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}
		return CopyFile(path, destPath)
	})
}
