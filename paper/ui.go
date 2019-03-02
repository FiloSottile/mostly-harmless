package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/andrew-d/go-termutil"
	"github.com/fatih/color"
)

func pickOutput() (outputW io.WriteCloser) {
	if len(os.Args) == 3 {
		f, err := os.Create(os.Args[2])
		fatalIfErr(err)
		return f
	}
	return os.Stdout
}

func getPass(msg string) []byte {
	msg = "[*] " + msg
	if stdinPwd {
		fmt.Fprint(os.Stderr, msg)
		passphrase, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
		fatalIfErr(err)
		return passphrase[:len(passphrase)-1]
	}
	tty := os.Stdin
	if stdinInput || !termutil.Isatty(tty.Fd()) {
		var err error
		tty, err = os.Open("/dev/tty")
		fatalIfErr(err)
	}
	passphrase, err := termutil.GetPass(msg, os.Stderr.Fd(), tty.Fd())
	fatalIfErr(err)
	return passphrase
}

var letterRe = regexp.MustCompile(`[a-zA-Z]`)
var wordsReader *bufio.Reader

func getWords() (words []string) {
	if wordsReader == nil {
		input := io.Reader(os.Stdin)
		if stdinInput {
			var err error
			input, err = os.Open("/dev/tty")
			fatalIfErr(err)
		}
		wordsReader = bufio.NewReader(input)
	}
	fmt.Fprint(os.Stderr, "[ ] ")
	line, err := wordsReader.ReadString('\n')
	fatalIfErr(err)
	for _, w := range strings.Split(line[:len(line)-1], " ") {
		if len(w) == 0 {
			continue
		}
		if letterRe.FindString(w) == "" {
			continue
		}
		words = append(words, strings.ToLower(w))
	}
	return
}

func fatalIfErr(err error) {
	if err != nil {
		logFatal("Error: %v", err)
	}
}

func logFatal(format string, a ...interface{}) {
	logError(format, a...)
	os.Exit(1)
}

func logError(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stderr, "[%s] %s\n", color.RedString("-"), msg)
}

func logInfo(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stderr, "[%s] %s\n", color.GreenString("+"), msg)
}

func printWords(words []string) {
	for n, w := range words {
		if n%10 == 0 {
			if n != 0 {
				fmt.Print("\n")
			}
			fmt.Printf("%2d: ", n/10+1)
		}
		fmt.Print(w, " ")
	}
	fmt.Print("\n")
}
