package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAssetListUnmarshal(t *testing.T) {
	var cfg Config
	err := yaml.Unmarshal([]byte(`
files:
  - ./plain.txt
  - path: ./explicit.db
    delivery: save
directories:
  - ./plain-dir
`), &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Files) != 2 || cfg.Files[0].Path != "./plain.txt" || cfg.Files[0].Delivery != DeliverySend {
		t.Fatalf("files: %+v", cfg.Files)
	}
	if cfg.Files[1].Delivery != DeliverySave {
		t.Fatalf("files[1] delivery: %s", cfg.Files[1].Delivery)
	}
	if len(cfg.Directories) != 1 || cfg.Directories[0].Path != "./plain-dir" {
		t.Fatalf("directories: %+v", cfg.Directories)
	}
}
