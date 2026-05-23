package backup

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func listPublicTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
		ORDER BY tablename`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func countTableRows(db *sql.DB, table string) (int64, error) {
	var n int64
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, quoteIdent(table))
	if err := db.QueryRow(query).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func collectDumpedTableStats(db *sql.DB, excluded map[string][]string) ([]TableRowCount, error) {
	tables, err := listPublicTables(db)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список таблиц: %w", err)
	}

	var stats []TableRowCount
	for _, table := range tables {
		if _, skip := excluded[table]; skip {
			continue
		}
		rows, err := countTableRows(db, table)
		if err != nil {
			return nil, fmt.Errorf("не удалось посчитать строки в %s: %w", table, err)
		}
		stats = append(stats, TableRowCount{Name: table, Rows: rows})
	}
	return stats, nil
}

func collectDirectoryStats(root, displayName string) (DirectoryReport, error) {
	var fileCount int
	var sizeBytes int64

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		fileCount++
		sizeBytes += info.Size()
		return nil
	})
	if err != nil {
		return DirectoryReport{}, fmt.Errorf("не удалось просканировать каталог %s: %w", root, err)
	}

	const bytesPerMB = 1024 * 1024
	return DirectoryReport{
		Name:      displayName,
		FileCount: fileCount,
		SizeMB:    float64(sizeBytes) / bytesPerMB,
	}, nil
}
