package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

var stdin = bufio.NewReader(os.Stdin)

func Input(title, current string) (string, error) {
	return editLine(title+": ", current, false)
}

func Password(title string) (string, error) {
	return editLine(title+": ", "", true)
}

func Credential() (userID, password string, err error) {
	if userID, err = Input("ID", ""); err != nil {
		return "", "", err
	}
	password, err = Password("パスワード")
	return userID, password, err
}

func Text(title, current string) (string, error) {
	fmt.Println(title + " (空行で終了):")
	initial := strings.Split(current, "\n")
	var lines []string
	for i := 0; ; i++ {
		line, err := editLine("  ", at(initial, i), false)
		if err != nil {
			return "", err
		}
		if line == "" {
			return strings.Join(lines, "\n"), nil
		}
		lines = append(lines, line)
	}
}

func Confirm(title string) (bool, error) {
	line, err := editLine(title+" (y/N) ", "", false)
	if err != nil {
		return false, err
	}
	return isYes(line), nil
}

func editLine(prompt, initial string, mask bool) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		fmt.Print(prompt)
		return readLine(stdin)
	}

	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, state)
	return edit(stdin, os.Stdout, prompt, initial, mask)
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if errors.Is(err, io.EOF) && line == "" {
		return "", ErrAborted
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func isYes(answer string) bool {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

func at(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}
