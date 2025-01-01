// Copyright 2014 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	flags   *flag.FlagSet
	verbose = new(count) // installed as -v below
	noRun   = new(bool)
)

const progName = "jj-codereview"

func initFlags() {
	flags = flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(stderr(), usage, progName, progName)
		exit(2)
	}
	flags.SetOutput(stderr())
	flags.BoolVar(noRun, "n", false, "print but do not run commands")
	flags.Var(verbose, "v", "report commands")
}

const globalFlags = "[-n] [-v]"

const usage = `Usage: %s <command> ` + globalFlags + `

Use "%s help" for a list of commands.
`

const help = `Usage: %s <command> ` + globalFlags + `

The -n flag prints commands that would make changes but does not run them.
The -v flag prints those commands as they run.

Available commands:

	mail [-r reviewer,...] [-cc mail,...] [options] [revisions]

`

func main() {
	initFlags()

	if len(os.Args) < 2 {
		flags.Usage()
		exit(2)
	}
	command, args := os.Args[1], os.Args[2:]

	// NOTE: Keep this switch in sync with the list of commands above.
	var cmd func([]string)
	switch command {
	default:
		flags.Usage()
		exit(2)
	case "help":
		fmt.Fprintf(stdout(), help, progName)
		return
	case "mail", "m":
		cmd = cmdMail
	}

	cmd(args)
}

func expectZeroArgs(args []string, command string) {
	flags.Parse(args)
	if len(flags.Args()) > 0 {
		fmt.Fprintf(stderr(), "Usage: %s %s %s\n", progName, command, globalFlags)
		exit(2)
	}
}

func setEnglishLocale(cmd *exec.Cmd) {
	// Override the existing locale to prevent non-English locales from
	// interfering with string parsing. See golang.org/issue/33895.
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, "LC_ALL=C")
}

func run(command string, args ...string) {
	if err := runErr(command, args...); err != nil {
		if *verbose == 0 {
			// If we're not in verbose mode, print the command
			// before dying to give context to the failure.
			fmt.Fprintf(stderr(), "(running: %s)\n", commandString(command, args))
		}
		dief("%v", err)
	}
}

func runErr(command string, args ...string) error {
	return runDirErr(".", command, args...)
}

var runLogTrap []string

func runDirErr(dir, command string, args ...string) error {
	if *noRun || *verbose == 1 {
		fmt.Fprintln(stderr(), commandString(command, args))
	} else if *verbose > 1 {
		start := time.Now()
		defer func() {
			fmt.Fprintf(stderr(), "%s # %.3fs\n", commandString(command, args), time.Since(start).Seconds())
		}()
	}
	if *noRun {
		return nil
	}
	if runLogTrap != nil {
		runLogTrap = append(runLogTrap, strings.TrimSpace(command+" "+strings.Join(args, " ")))
	}
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout()
	cmd.Stderr = stderr()
	if dir != "." {
		cmd.Dir = dir
	}
	setEnglishLocale(cmd)
	return cmd.Run()
}

// cmdOutput runs the command line, returning its output.
// If the command cannot be run or does not exit successfully,
// cmdOutput dies.
//
// NOTE: cmdOutput must be used only to run commands that read state,
// not for commands that make changes. Commands that make changes
// should be run using runDirErr so that the -v and -n flags apply to them.
func cmdOutput(command string, args ...string) string {
	return cmdOutputDir(".", command, args...)
}

// cmdOutputDir runs the command line in dir, returning its output.
// If the command cannot be run or does not exit successfully,
// cmdOutput dies.
//
// NOTE: cmdOutput must be used only to run commands that read state,
// not for commands that make changes. Commands that make changes
// should be run using runDirErr so that the -v and -n flags apply to them.
func cmdOutputDir(dir, command string, args ...string) string {
	s, err := cmdOutputDirErr(dir, command, args...)
	if err != nil {
		fmt.Fprintf(stderr(), "%v\n%s\n", commandString(command, args), s)
		dief("%v", err)
	}
	return s
}

// cmdOutputErr runs the command line in dir, returning its output
// and any error results.
//
// NOTE: cmdOutputErr must be used only to run commands that read state,
// not for commands that make changes. Commands that make changes
// should be run using runDirErr so that the -v and -n flags apply to them.
func cmdOutputErr(command string, args ...string) (string, error) {
	return cmdOutputDirErr(".", command, args...)
}

// cmdOutputDirErr runs the command line in dir, returning its output
// and any error results.
//
// NOTE: cmdOutputDirErr must be used only to run commands that read state,
// not for commands that make changes. Commands that make changes
// should be run using runDirErr so that the -v and -n flags apply to them.
func cmdOutputDirErr(dir, command string, args ...string) (string, error) {
	// NOTE: We only show these non-state-modifying commands with -v -v.
	// Otherwise things like 'git sync -v' show all our internal "find out about
	// the git repo" commands, which is confusing if you are just trying to find
	// out what git sync means.
	if *verbose > 1 {
		start := time.Now()
		defer func() {
			fmt.Fprintf(stderr(), "%s # %.3fs\n", commandString(command, args), time.Since(start).Seconds())
		}()
	}
	cmd := exec.Command(command, args...)
	if dir != "." {
		cmd.Dir = dir
	}
	setEnglishLocale(cmd)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

// trim is shorthand for strings.TrimSpace.
func trim(text string) string {
	return strings.TrimSpace(text)
}

// trimErr applies strings.TrimSpace to the result of cmdOutput(Dir)Err,
// passing the error along unmodified.
func trimErr(text string, err error) (string, error) {
	return strings.TrimSpace(text), err
}

// lines returns the lines in text.
func lines(text string) []string {
	out := strings.Split(text, "\n")
	// Split will include a "" after the last line. Remove it.
	if n := len(out) - 1; n >= 0 && out[n] == "" {
		out = out[:n]
	}
	return out
}

// nonBlankLines returns the non-blank lines in text.
func nonBlankLines(text string) []string {
	var out []string
	for _, s := range lines(text) {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func commandString(command string, args []string) string {
	return strings.Join(append([]string{command}, args...), " ")
}

func dief(format string, args ...interface{}) {
	printf(format, args...)
	exit(1)
}

var exitTrap func()

func exit(code int) {
	if exitTrap != nil {
		exitTrap()
	}
	os.Exit(code)
}

func verbosef(format string, args ...interface{}) {
	if *verbose > 0 {
		printf(format, args...)
	}
}

var stdoutTrap, stderrTrap *bytes.Buffer

func stdout() io.Writer {
	if stdoutTrap != nil {
		return stdoutTrap
	}
	return os.Stdout
}

func stderr() io.Writer {
	if stderrTrap != nil {
		return stderrTrap
	}
	return os.Stderr
}

func printf(format string, args ...interface{}) {
	lines := strings.Split(fmt.Sprintf(format, args...), "\n")
	for _, line := range lines {
		fmt.Fprintf(stderr(), "%s: %s\n", progName, line)
	}
}

// count is a flag.Value that is like a flag.Bool and a flag.Int.
// If used as -name, it increments the count, but -name=x sets the count.
// Used for verbose flag -v.
type count int

func (c *count) String() string {
	return fmt.Sprint(int(*c))
}

func (c *count) Set(s string) error {
	switch s {
	case "true":
		*c++
	case "false":
		*c = 0
	default:
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid count %q", s)
		}
		*c = count(n)
	}
	return nil
}

func (c *count) IsBoolFlag() bool {
	return true
}
