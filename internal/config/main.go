package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type DumpConfig struct {
	Host     string   `yaml:"host"`     // адрес сервера
	Port     string   `yaml:"port"`     // порт
	User     string   `yaml:"user"`     // пользователь
	Password string   `yaml:"password"` // пароль
	DBName   string   `yaml:"name"`     // имя базы
	Exclude  []string `yaml:"exclude"`  // исключаемые таблицы
}

type Config struct {
	Databases []DumpConfig `yaml:"databases"`

	Directories []string `yaml:"directories"`
	Files       []string `yaml:"files"`

	Telegram struct {
		Token  string `yaml:"token"`
		ChatID string `yaml:"chat_id"`
	} `yaml:"telegram"`

	DumpDir string `yaml:"dump_dir"`
}

func Read() (*Config, error) {
	cfg := Config{}

	data, err := os.ReadFile("config.yml")
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга YAML: %w", err)
	}

	return &cfg, nil
}
