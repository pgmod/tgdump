package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Delivery string

const (
	DeliverySave Delivery = "save" // только локальный архив
	DeliverySend Delivery = "send" // локальный архив и Telegram
)

func (d Delivery) ShouldSend() bool {
	return NormalizeDelivery(d) == DeliverySend
}

func (d Delivery) Label() string {
	if d.ShouldSend() {
		return "сохранение и отправка"
	}
	return "только сохранение"
}

func NormalizeDelivery(d Delivery) Delivery {
	switch d {
	case DeliverySave, DeliverySend:
		return d
	default:
		return DeliverySend
	}
}

type AssetEntry struct {
	Path     string   `yaml:"path"`
	Delivery Delivery `yaml:"delivery"`
}

type AssetList []AssetEntry

func (l *AssetList) UnmarshalYAML(node *yaml.Node) error {
	var items []yaml.Node
	if err := node.Decode(&items); err != nil {
		return err
	}

	entries := make([]AssetEntry, 0, len(items))
	for i, item := range items {
		var entry AssetEntry
		switch item.Kind {
		case yaml.ScalarNode:
			entry.Path = item.Value
			entry.Delivery = DeliverySend
		case yaml.MappingNode:
			if err := item.Decode(&entry); err != nil {
				return fmt.Errorf("files/directories[%d]: %w", i, err)
			}
			entry.Delivery = NormalizeDelivery(entry.Delivery)
			if entry.Path == "" {
				return fmt.Errorf("files/directories[%d]: path is required", i)
			}
		default:
			return fmt.Errorf("files/directories[%d]: unexpected YAML node", i)
		}
		entries = append(entries, entry)
	}
	*l = entries
	return nil
}
