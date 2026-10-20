package ui

import (
	"testing"
	"time"
)

func TestDeadline(t *testing.T) {
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.Local)
	for name, tc := range map[string]struct {
		end  time.Time
		want string
	}{
		"minutes":  {now.Add(45 * time.Minute), "10/20 10:45 締切 (あと 45 分)"},
		"hours":    {now.Add(7*time.Hour + 30*time.Minute), "10/20 17:30 締切 (あと 7 時間)"},
		"days":     {now.Add(3*24*time.Hour + 7*time.Hour), "10/23 17:00 締切 (あと 3 日)"},
		"past":     {now.Add(-time.Hour), "10/20 09:00 締切 (締切済み)"},
		"no limit": {time.Time{}, "締切なし"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := Deadline(tc.end, now); got != tc.want {
				t.Errorf("Deadline = %q, want %q", got, tc.want)
			}
		})
	}
}
