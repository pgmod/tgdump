package backup

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"tgdump/internal/config"
)

func pgEnv(password string) []string {
	return append(os.Environ(), "PGPASSWORD="+password)
}

func runPsql(cfg config.DumpConfig, query string) (string, error) {
	cmd := exec.Command("psql",
		"-h", cfg.Host,
		"-p", cfg.Port,
		"-U", cfg.User,
		"-d", cfg.DBName,
		"-c", query,
	)
	cmd.Env = pgEnv(cfg.Password)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("ошибка psql: %w, вывод: %s", err, out.String())
	}
	return out.String(), nil
}

func runPgDump(cfg config.DumpConfig, args ...string) error {
	cmd := exec.Command("pg_dump", args...)
	cmd.Env = pgEnv(cfg.Password)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка выполнения pg_dump: %w, output: %s", err, string(output))
	}
	return nil
}

func dropTempTables(cfg config.DumpConfig, tables map[string][]string) {
	for table := range tables {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s_temp;", table)
		if _, err := runPsql(cfg, query); err != nil {
			fmt.Printf("ошибка удаления временной таблицы %s: %v\n", table, err)
		}
	}
}
