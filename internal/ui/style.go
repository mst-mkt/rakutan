package ui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
)

var (
	colored     = os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(os.Stdout.Fd()))
	styleEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

func stripStyle(s string) string {
	return styleEscape.ReplaceAllString(s, "")
}

func style(codes string) func(string) string {
	return func(s string) string {
		if !colored || s == "" {
			return s
		}
		return "\x1b[" + codes + "m" + s + "\x1b[0m"
	}
}

var (
	bold      = style("1")
	faint     = style("2")
	heading   = style("1;4")
	cyan      = style("36")
	green     = style("32")
	yellow    = style("33")
	red       = style("31")
	promptBar = style("1;36")
)

func promptLine(title, value string) string {
	return promptBar("?") + " " + bold(title) + ": " + value
}

func Warn(msg string, err error) {
	fmt.Fprintf(os.Stderr, "%s %s: %v\n", yellow("warning:"), msg, err)
}

func Error(err error) {
	fmt.Fprintf(os.Stderr, "%s %v\n", red(bold("error:")), err)
}

func Notice(msg string) {
	fmt.Fprintln(os.Stderr, yellow(msg))
}

func Done(msg string) {
	fmt.Println(green(msg))
}

func markColor(mark string) func(string) string {
	switch {
	case mark == "○" || strings.HasPrefix(mark, "100%"):
		return green
	case mark == "×" || strings.HasPrefix(mark, "0%"):
		return red
	default:
		return yellow
	}
}

func statusColor(status string) func(string) string {
	switch {
	case strings.Contains(status, "提出済み"):
		return green
	case strings.Contains(status, "受付中"):
		return yellow
	default:
		return faint
	}
}
