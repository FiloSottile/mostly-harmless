package main

import (
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

var pgpArmor bool

func main() {
	input, err := ioutil.ReadAll(os.Stdin)
	fatalIfErr(err)
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
		logInfo("PGP private key detected, generating backup codes")
		inputR.Seek(0, 0)
		pgpBackup(inputR)

	default:
		logFatal("Unrecognized PGP packet: %T", p)
	}
}

func pgpDecrypt(p *packet.PrivateKey, passphrase []byte) []byte {
	if err := p.Decrypt(passphrase); err == nil {
		return passphrase
	}
	tty := os.Stdin
	var err error
	if !termutil.Isatty(tty.Fd()) {
		tty, err = os.Open("/dev/tty")
		fatalIfErr(err)
	}
	passphrase, err = termutil.GetPass(
		"[*] Enter passphrase for PGP key "+p.KeyIdShortString()+": ",
		os.Stderr.Fd(), tty.Fd(),
	)
	fatalIfErr(err)
	if p.Decrypt(passphrase) != nil {
		logFatal("Decryption failed, is the passphrase right?")
	}
	logInfo("Decryption successful")
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
