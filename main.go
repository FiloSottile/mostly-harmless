package main

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"regexp"
	"strings"

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
	inputR.Seek(0, 0)

	switch p := p.(type) {
	case *packet.PrivateKey:
		if len(os.Args) != 2 {
			logFatal("Can't specify OUTPUT file when generating backups")
		}
		logInfo("PGP private key detected, generating backup codes")
		pgpBackup(inputR)

	case *packet.PublicKey:
		var outputW io.WriteCloser
		switch {
		case len(os.Args) == 2:
			outputW = os.Stdout
		case len(os.Args) == 3:
			f, err := os.Create(os.Args[2])
			fatalIfErr(err)
			outputW = f
		}
		logInfo("PGP public key detected, regenerating private key")
		pgpRestore(inputR, outputW)

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

func pgpRestore(inputR *bytes.Reader, outputW io.WriteCloser) {
	r := packet.NewOpaqueReader(inputR)
	for {
		op, err := r.Next()
		if err == io.EOF {
			break
		}
		fatalIfErr(err)
		p, err := op.Parse()
		if err != nil {
			fatalIfErr(op.Serialize(outputW))
			continue
		}
		pk, ok := p.(*packet.PublicKey)
		if !ok {
			fatalIfErr(op.Serialize(outputW))
			continue
		}

		var priv *rsa.PrivateKey
		switch key := pk.PublicKey.(type) {
		case *rsa.PublicKey:
			logInfo("Restoring key %s, please type the backup words", pk.KeyIdShortString())
			logInfo("You can start a new line at any time, and words will be spell checked")
			var words []string
			for {
				newWords := getWords()
				_, corr, wrong := Bip39Decode(newWords)
				if len(wrong) != 0 {
					logError("Words not recognized (entire line was discarded): %s", strings.Join(wrong, ", "))
					continue
				}
				if len(corr) != 0 {
					logInfo("Words autocorrected (all %d words accepted): %s",
						len(newWords), strings.Join(corr, ", "))
				} else {
					logInfo("%d words accepted", len(newWords))
				}
				words = append(words, newWords...)
				data, _, _ := Bip39Decode(words)
				priv, err = TryRSAKey(key, data)
				fatalIfErr(err)
				if priv == nil {
					continue
				}
				privKey := &packet.PrivateKey{
					PublicKey:  *pk,
					PrivateKey: priv,
				}
				fatalIfErr(privKey.Serialize(outputW))
				logInfo("Private key successfully recovered!")
				break
			}

		default:
			logFatal("Unsupported key algorithm: %T", key)
		}
	}
}

func TryRSAKey(pub *rsa.PublicKey, data []byte) (*rsa.PrivateKey, error) {
	q := new(big.Int).SetBytes(data)
	if q.BitLen() > pub.N.BitLen()/2+8 {
		return nil, errors.New("words sequence got too long with no match")
	}
	if new(big.Int).Rem(pub.N, q).BitLen() != 0 {
		return nil, nil
	}

	p := new(big.Int).Quo(pub.N, q)
	priv := &rsa.PrivateKey{
		PublicKey: *pub,
		Primes:    []*big.Int{p, q},
		D:         new(big.Int),
	}

	totient := big.NewInt(1)
	pminus1 := new(big.Int)
	for _, prime := range priv.Primes {
		pminus1.Sub(prime, big.NewInt(1))
		totient.Mul(totient, pminus1)
	}
	new(big.Int).GCD(priv.D, nil, big.NewInt(int64(pub.E)), totient)
	if priv.D.Sign() < 0 {
		priv.D.Add(priv.D, totient)
	}

	priv.Precompute()
	if err := priv.Validate(); err != nil {
		return nil, err
	}
	return priv, nil
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

func printFooter(numWords int) {
	fmt.Fprint(os.Stderr, "\n")
	logInfo("You will be able to regenerate the secret key by running this")
	logInfo("tool again on the public key and typing the provided %d words", numWords)
	logInfo("Testing the restore process is highly recommended")
}
