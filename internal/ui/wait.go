package ui

import (
	"fmt"
	"time"
)

func Wait(message string, f func() error) error {
	done := make(chan error, 1)
	go func() { done <- f() }()

	start := time.Now()
	show := func() {
		fmt.Printf("\r%s %s\x1b[K", faint(message+"…"), faint(fmt.Sprintf("%d 秒", int(time.Since(start).Seconds()))))
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	show()
	for {
		select {
		case err := <-done:
			show()
			fmt.Println()
			return err
		case <-ticker.C:
			show()
		}
	}
}
