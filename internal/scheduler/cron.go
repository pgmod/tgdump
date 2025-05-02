package scheduler

import (
	"fmt"
	"strings"

	"github.com/robfig/cron/v3"
)

// ScheduleDailyAt запускает функцию каждый день в указанное время (например, "08:00")
func ScheduleDailyAt(timeStr string, job func()) {
	c := cron.New()

	// Парсим время: "08:00" → 0 0 8 * * *
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		panic("неправильный формат времени, должен быть HH:MM")
	}

	spec := fmt.Sprintf("%s %s * * *", parts[1], parts[0]) // секунда, минута, час

	_, err := c.AddFunc(spec, job)
	if err != nil {
		panic(fmt.Sprintf("не удалось добавить задачу в cron: %v", err))
	}

	c.Start()
}
