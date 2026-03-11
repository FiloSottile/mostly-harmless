package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

const (
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorDim   = "\033[2m"
)

func colorize(tty bool, s, color string) string {
	if !tty || s == "" {
		return s
	}
	return fmt.Sprintf("%s%s\033[0m", color, s)
}

func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
