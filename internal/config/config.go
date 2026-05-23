package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	defaultFilesDir = "./files"
	defaultSchedule = "08:00"
)

type DumpConfig struct {
	Host     string   `yaml:"host"`
	Port     string   `yaml:"port"`
	User     string   `yaml:"user"`
	Password string   `yaml:"password"`
	DBName   string   `yaml:"name"`
	Exclude  []string `yaml:"exclude"`
	Delivery Delivery `yaml:"delivery"`
}

type Config struct {
	Databases   []DumpConfig `yaml:"databases"`
	Directories AssetList    `yaml:"directories"`
	Files       AssetList    `yaml:"files"`
	FilesDir    string       `yaml:"files_dir"`

	Telegram struct {
		Token  string `yaml:"token"`
		ChatID string `yaml:"chat_id"`
	} `yaml:"telegram"`

	DumpDir  string `yaml:"dump_dir"`
	Schedule string `yaml:"schedule"`
}

func (c *Config) Print() {
	fmt.Println("Databases:")
	for _, db := range c.Databases {
		fmt.Printf("  - %s [%s]\n", db.DBName, db.Delivery.Label())
	}
	fmt.Println("Directories:")
	for _, dir := range c.Directories {
		fmt.Printf("  - %s [%s]\n", dir.Path, dir.Delivery.Label())
	}
	fmt.Println("Files:")
	for _, file := range c.Files {
		fmt.Printf("  - %s [%s]\n", file.Path, file.Delivery.Label())
	}
	fmt.Println("FilesDir:")
	fmt.Printf("  - %s\n", c.FilesDir)
	fmt.Println("Telegram:")
	fmt.Printf("  - ChatID: %s\n", c.Telegram.ChatID)
	fmt.Println("DumpDir:")
	fmt.Printf("  - %s\n", c.DumpDir)
	fmt.Println("Schedule:")
	fmt.Printf("  - %s\n", c.Schedule)
}

func Read() (*Config, error) {
	data, err := os.ReadFile("config.yml")
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга YAML: %w", err)
	}

	normalizeConfig(&cfg)
	return &cfg, nil
}

func normalizeConfig(cfg *Config) {
	if cfg.FilesDir == "" {
		cfg.FilesDir = defaultFilesDir
	}
	if cfg.Schedule == "" {
		cfg.Schedule = defaultSchedule
	}
	for i := range cfg.Databases {
		cfg.Databases[i].Delivery = NormalizeDelivery(cfg.Databases[i].Delivery)
	}
	for i := range cfg.Files {
		cfg.Files[i].Delivery = NormalizeDelivery(cfg.Files[i].Delivery)
	}
	for i := range cfg.Directories {
		cfg.Directories[i].Delivery = NormalizeDelivery(cfg.Directories[i].Delivery)
	}
}
