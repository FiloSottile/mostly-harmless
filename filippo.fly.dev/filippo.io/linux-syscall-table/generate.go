package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

func main() {
	const Version = "6.7"

	resp, err := http.Get("https://raw.githubusercontent.com/torvalds/linux/v" + Version + "/arch/x86/entry/syscalls/syscall_64.tbl")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	table, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	resp, err = http.Get("https://raw.githubusercontent.com/torvalds/linux/v" + Version + "/include/linux/syscalls.h")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	headers, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	type syscall struct {
		Number      string
		Name        string
		Implemented bool
		Entrypoint  string
		Args        []string
	}
	var syscalls []syscall

	lines := strings.Split(string(table), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if parts[1] == "x32" {
			continue
		}

		number := parts[0]
		name := parts[2]

		if len(parts) < 4 {
			syscalls = append(syscalls, syscall{Number: number, Name: name})
			continue
		}

		entry := parts[3]
		if entry == "sys_mmap" {
			entry = "ksys_mmap_pgoff"
		}

		re := regexp.MustCompile(`(?:asmlinkage|unsigned) long ` + entry + `\(([^)]+)\);`)
		matches := re.FindStringSubmatch(string(headers))
		var args []string
		switch {
		case matches != nil && matches[1] == "void":
			args = []string{}
		case matches != nil:
			args = strings.Split(matches[1], ",")
			for i, arg := range args {
				args[i] = strings.TrimSpace(arg)
				args[i] = strings.ReplaceAll(args[i], "__user ", "")
			}
		case entry == "sys_rt_sigreturn":
			args = []string{}
		case entry == "sys_modify_ldt":
			args = []string{"int func", "void *ptr", "unsigned long bytecount"}
		case entry == "sys_arch_prctl":
			args = []string{"int option", "unsigned long arg2"}
		case entry == "sys_iopl":
			args = []string{"unsigned int level"}
		default:
			panic("no match for " + name)
		}

		syscalls = append(syscalls, syscall{
			Number: number, Name: name, Implemented: true,
			Entrypoint: strings.TrimPrefix(entry, "sys_"), Args: args})
	}

	// Load the HTML template from file index.html.tmpl
	tmpl, err := template.ParseFiles("index.html.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	if err := tmpl.Execute(os.Stdout, map[string]interface{}{
		"Version":   Version,
		"Syscalls":  syscalls,
		"Registers": []string{"rdi", "rsi", "rdx", "r10", "r8", "r9"},
	}); err != nil {
		log.Fatal(err)
	}
}
