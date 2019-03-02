// Command paper generates short, easy to write down backups of private keys.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

var (
	// INPUT is -, don't try to read passwords from stdin
	stdinInput bool

	// Read passwords from stdin even if it's not a TTY (used for testing)
	stdinPwd bool
)

func usage() {
	fmt.Println(`Usage: paper INPUT [OUTPUT]

INPUT is either a public or a private PGP key (password protected and ASCII
armored keys are supported). To read from standard input specify "-".

Private keys will be converted to paper-friendly backups, public keys will be
reconstructed into private keys once the backup words are entered.

If private keys are generated, they will written to OUTPUT. If OUTPUT is
omitted private keys will be written to standard output.`)
	os.Exit(3)
}

func main() {
	if len(os.Args) != 2 && len(os.Args) != 3 {
		usage()
	}

	var input []byte
	if os.Args[1] == "-" {
		stdinInput = true
		var err error
		input, err = ioutil.ReadAll(os.Stdin)
		fatalIfErr(err)
	} else {
		f, err := os.Open(os.Args[1])
		fatalIfErr(err)
		input, err = ioutil.ReadAll(f)
		fatalIfErr(err)
	}

	if tryPGP(input) {
		return
	}

	logFatal("Input not recognized. Supported formats: PGP (armored and not, encrypted and not), SSH (encrypted and not)")
}
