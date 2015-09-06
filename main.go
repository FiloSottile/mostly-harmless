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

func fatalIfErr(err error) {
	if err != nil {
		logFatal("Error: " + err.Error())
	}
}

func logFatal(msg string) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", color.RedString("-"), msg)
	os.Exit(1)
}

func logInfo(msg string) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", color.GreenString("+"), msg)
}

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
		logFatal(fmt.Sprintf("Unrecognized PGP packet: %T", p))
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
	logInfo("Decryption succeeded")
	return passphrase
}

func pgpBackup(inputR *bytes.Reader) {
	var passphrase = []byte("")
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
			Bip39Encode(key.Primes[0].Bytes())
		default:
			logFatal(fmt.Sprintf("Unsupported key algorithm: %T", key))
		}
	}
}
