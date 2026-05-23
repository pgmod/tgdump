package backup

import (
	"database/sql"
	"fmt"
	"strings"

	"tgdump/internal/config"

	_ "github.com/lib/pq"
)

func parseExcludes(excludes []string) map[string][]string {
	excludeMap := make(map[string][]string)
	for _, item := range excludes {
		parts := strings.Split(item, ".")
		if len(parts) != 2 {
			continue
		}
		table, column := parts[0], parts[1]
		excludeMap[table] = append(excludeMap[table], column)
	}
	return excludeMap
}

func getColumnsExcluding(db *sql.DB, table string, excludeCols []string) ([]string, error) {
	excludeSet := make(map[string]struct{}, len(excludeCols))
	for _, col := range excludeCols {
		excludeSet[col] = struct{}{}
	}

	rows, err := db.Query(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, err
		}
		if _, excluded := excludeSet[col]; !excluded {
			columns = append(columns, col)
		}
	}
	return columns, rows.Err()
}

func prepareTempTable(cfg config.DumpConfig, table string, columns []string) error {
	query := fmt.Sprintf(`
		DROP TABLE IF EXISTS %s_temp;
		CREATE TABLE %s_temp AS SELECT %s FROM %s;`,
		table, table, strings.Join(columns, ", "), table)
	_, err := runPsql(cfg, query)
	return err
}

func DumpDatabaseEx(cfg config.DumpConfig, outFile string) ([]TableRowCount, error) {
	excludeMap := parseExcludes(cfg.Exclude)

	dbinfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)
	db, err := sql.Open("postgres", dbinfo)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе: %w", err)
	}
	defer db.Close()

	for table, excludedCols := range excludeMap {
		cols, err := getColumnsExcluding(db, table, excludedCols)
		if err != nil {
			return nil, fmt.Errorf("не удалось получить колонки таблицы %s: %w", table, err)
		}
		if err := prepareTempTable(cfg, table, cols); err != nil {
			return nil, err
		}
	}
	defer dropTempTables(cfg, excludeMap)

	stats, err := collectDumpedTableStats(db, excludeMap)
	if err != nil {
		return nil, err
	}

	args := []string{
		"-h", cfg.Host,
		"-p", cfg.Port,
		"-U", cfg.User,
		"-F", "p",
		"-f", outFile,
	}
	for table := range excludeMap {
		args = append(args, fmt.Sprintf("--exclude-table=public.%s", table))
	}
	args = append(args, cfg.DBName)

	if err := runPgDump(cfg, args...); err != nil {
		return nil, err
	}
	return stats, nil
}
