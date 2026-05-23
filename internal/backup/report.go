package backup

import (
	"fmt"
	"strings"

	"tgdump/internal/config"
)

type TableRowCount struct {
	Name string
	Rows int64
}

type DatabaseReport struct {
	Name     string
	Delivery config.Delivery
	Tables   []TableRowCount
}

type DirectoryReport struct {
	Name      string
	Delivery  config.Delivery
	FileCount int
	SizeMB    float64
}

type FileReport struct {
	Name     string
	Delivery config.Delivery
}

type Report struct {
	Timestamp   string
	Databases   []DatabaseReport
	Directories []DirectoryReport
	Files       []FileReport
}

func (r Report) Format() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Резервная копия: %s\n", r.Timestamp)
	for _, db := range r.Databases {
		fmt.Fprintf(&b, "\nБаза %s [%s]:\n", db.Name, db.Delivery.Label())
		for _, t := range db.Tables {
			fmt.Fprintf(&b, "  %s: %d\n", t.Name, t.Rows)
		}
	}
	if len(r.Files) > 0 {
		b.WriteString("\nФайлы:\n")
		for _, f := range r.Files {
			fmt.Fprintf(&b, "  %s [%s]\n", f.Name, f.Delivery.Label())
		}
	}
	if len(r.Directories) > 0 {
		b.WriteString("\nКаталоги:\n")
		for _, d := range r.Directories {
			fmt.Fprintf(&b, "  %s: %d файлов, %.2f МБ [%s]\n", d.Name, d.FileCount, d.SizeMB, d.Delivery.Label())
		}
	}
	return b.String()
}
