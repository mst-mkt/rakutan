package ui

import (
	"fmt"
	"time"
)

const (
	urgent = 24 * time.Hour
	soon   = 3 * 24 * time.Hour
)

func Deadline(end, now time.Time) string {
	if end.IsZero() {
		return "締切なし"
	}
	left := end.Sub(now)
	text := fmt.Sprintf("%s 締切 (%s)", end.Format("01/02 15:04"), remaining(left))
	switch {
	case left < urgent:
		return red(text)
	case left < soon:
		return yellow(text)
	default:
		return text
	}
}

func remaining(d time.Duration) string {
	switch {
	case d <= 0:
		return "締切済み"
	case d < time.Hour:
		return fmt.Sprintf("あと %d 分", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("あと %d 時間", int(d.Hours()))
	default:
		return fmt.Sprintf("あと %d 日", int(d.Hours()/24))
	}
}
