package main

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/andrew-d/go-termutil"
	"github.com/fatih/color"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

var (
	// The input file was armored, so should be the output file
	pgpArmor bool

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
omitted private keys will be written to standard output.
`)
	os.Exit(3)
}

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
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
	inputR := bytes.NewReader(input)

	b, err := armor.Decode(inputR)
	switch {
	case err != nil:
		// not a PGP armored input
		inputR.Seek(0, 0)

	case b.Type == "PGP PUBLIC KEY BLOCK" || b.Type == "PGP PRIVATE KEY BLOCK":
		pgpArmor = true
		logInfo("PGP armor encoded block detected")
		body, err := ioutil.ReadAll(b.Body)
		fatalIfErr(err)
		input, inputR = body, bytes.NewReader(body)

	default:
		logFatal("Unrecognized type: " + b.Type)
	}

	p, err := packet.Read(inputR)
	if err != nil || p == nil {
		logFatal("Couldn't detect any PGP packets")
	}

	switch p := p.(type) {
	case *packet.PrivateKey:
		if len(os.Args) > 2 {
			logFatal("Can't specify OUTPUT file when generating backups")
		}
		logInfo("PGP private key detected, generating backup codes")
		inputR.Seek(0, 0)
		pgpBackup(inputR)

	default:
		logFatal("Unrecognized PGP packet: %T", p)
	}
}

func pgpDecrypt(p *packet.PrivateKey, passphrase []byte) []byte {
	err := p.Decrypt(passphrase)
	if err == nil {
		return passphrase
	}
	passphrase = getPass("Enter passphrase for PGP key " + p.KeyIdShortString() + ": ")
	if p.Decrypt(passphrase) != nil {
		logFatal("Decryption failed, is the passphrase right?")
	}
	logInfo("Decryption successful")
	return passphrase
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

func pgpBackup(inputR *bytes.Reader) {
	var passphrase = []byte("")
	var numWords int
	r := packet.NewReader(inputR)
	for {
		p, err := r.Next()
		if err == io.EOF {
			break
		}
		fatalIfErr(err)
		pk, ok := p.(*packet.PrivateKey)
		if !ok {
			continue
		}

		if pk.Encrypted {
			passphrase = pgpDecrypt(pk, passphrase)
		}

		switch key := pk.PrivateKey.(type) {
		case *rsa.PrivateKey:
			if len(key.Primes) != 2 {
				logFatal("Unsupported number of primes")
			}
			logInfo("Generating backup sequence for key " + pk.KeyIdShortString())
			words := Bip39Encode(key.Primes[0].Bytes())
			printWords(words)
			numWords += len(words)
		default:
			logFatal("Unsupported key algorithm: %T", key)
		}
	}
	logInfo("Backup successful")
	printFooter(numWords)
}

func fatalIfErr(err error) {
	if err != nil {
		logFatal("Error: %v", err)
	}
}

func logFatal(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stderr, "[%s] %s\n", color.RedString("-"), msg)
	os.Exit(1)
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

func printFooter(numWords int) {
	fmt.Fprint(os.Stderr, "\n")
	logInfo("You will be able to regenerate the secret key by running this")
	logInfo("tool again on the public key and typing the provided %d words", numWords)
	logInfo("Testing the restore process is highly recommended")
}
