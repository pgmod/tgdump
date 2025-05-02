package backup

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"tgdump/internal/config"

	_ "github.com/lib/pq"
)

// DumpConfig содержит настройки для создания дампа

// DumpDatabase вызывает pg_dump и сохраняет дамп в указанный файл
func DumpDatabase(cfg config.DumpConfig, outFile string) error {
	// Устанавливаем переменную окружения для пароля
	os.Setenv("PGPASSWORD", cfg.Password)

	// Формируем команду
	cmd := exec.Command("pg_dump",
		"-h", cfg.Host,
		"-p", cfg.Port,
		"-U", cfg.User,
		"-F", "c", // custom формат (сжатый)
		"-f", outFile,
		cfg.DBName,
	)

	// Указываем, что команда должна выполняться в shell с нужным пользователем
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка выполнения pg_dump: %v, output: %s", err, string(output))
	}

	return nil
}

// разбираем список исключений в map[table][]column
func parseExcludes(excludes []string) map[string][]string {
	excludeMap := make(map[string][]string)
	for _, item := range excludes {
		parts := strings.Split(item, ".")
		if len(parts) != 2 {
			continue
		}
		table := parts[0]
		column := parts[1]
		excludeMap[table] = append(excludeMap[table], column)
	}
	return excludeMap
}

// получаем список колонок таблицы, исключая нужные
func getColumnsExcluding(db *sql.DB, table string, excludeCols []string) ([]string, error) {
	excludeSet := make(map[string]struct{})
	for _, col := range excludeCols {
		excludeSet[col] = struct{}{}
	}

	query := `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position`
	rows, err := db.Query(query, table)
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
	return columns, nil
}

// создаёт временную таблицу без определённых колонок
func prepareTempTable(cfg config.DumpConfig, table string, columns []string) error {
	columnList := strings.Join(columns, ", ")
	query := fmt.Sprintf(`
		DROP TABLE IF EXISTS %s_temp;
		CREATE TABLE %s_temp AS SELECT %s FROM %s;`,
		table, table, columnList, table)

	cmd := exec.Command("psql",
		"-h", cfg.Host,
		"-p", cfg.Port,
		"-U", cfg.User,
		"-d", cfg.DBName,
		"-c", query)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.Password))

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка создания временной таблицы: %v, вывод: %s", err, out.String())
	}
	return nil
}
func DumpDatabaseEx(cfg config.DumpConfig, outFile string) error {
	os.Setenv("PGPASSWORD", cfg.Password)

	excludeMap := parseExcludes(cfg.Exclude)

	dbinfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)
	db, err := sql.Open("postgres", dbinfo)
	if err != nil {
		return fmt.Errorf("ошибка подключения к базе: %v", err)
	}
	defer db.Close()

	// создаём временные таблицы без указанных колонок
	for table, excludedCols := range excludeMap {
		cols, err := getColumnsExcluding(db, table, excludedCols)
		if err != nil {
			return fmt.Errorf("не удалось получить колонки таблицы %s: %v", table, err)
		}
		if err := prepareTempTable(cfg, table, cols); err != nil {
			return err
		}
	}
	defer func() {
		for table := range excludeMap {
			fmt.Println("удаляем временную таблицу:", table)
			query := fmt.Sprintf("DROP TABLE IF EXISTS %s_temp;", table)
			cmd := exec.Command("psql", "-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DBName, "-c", query)
			cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.Password))
			err := cmd.Run()
			if err != nil {
				fmt.Println("ошибка удаления временной таблицы:", err)
			}
		}
	}()
	excludesCmd := make([]string, 0)
	for table := range excludeMap {
		excludesCmd = append(excludesCmd, fmt.Sprintf("--exclude-table=public.%s", table))
	}

	cmdArgs := []string{
		"-h", cfg.Host,
		"-p", cfg.Port,
		"-U", cfg.User,
		"-F", "p",
		"-f", outFile,
	}
	cmdArgs = append(cmdArgs, excludesCmd...)
	cmdArgs = append(cmdArgs, cfg.DBName)
	fmt.Println(cmdArgs)

	// делаем дамп всех таблиц (включая временные)
	cmd := exec.Command("pg_dump", cmdArgs...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка выполнения pg_dump: %v, output: %s", err, string(output))
	}

	return nil
}
