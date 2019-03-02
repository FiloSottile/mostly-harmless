package main

import (
	"bytes"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"strings"

	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

var (
	// The input file was armored, so should be the output file
	pgpArmor bool
)

func tryPGP(input []byte) bool {
	b, err := armor.Decode(bytes.NewReader(input))
	switch {
	case err != nil: // not a PGP armored input

	case b.Type == "PGP PUBLIC KEY BLOCK":
		logInfo("PGP armor encoded block detected, output will be armored")
		fallthrough
	case b.Type == "PGP PRIVATE KEY BLOCK":
		pgpArmor = true
		input, err = ioutil.ReadAll(b.Body)
		fatalIfErr(err)

	default:
		logFatal("Unrecognized type: " + b.Type)
	}

	p, err := packet.Read(bytes.NewReader(input))
	if err != nil || p == nil {
		if pgpArmor {
			logFatal("Corrupted PGP packet in valid armor")
		}
		return false
	}

	switch p := p.(type) {
	case *packet.PrivateKey:
		if len(os.Args) != 2 {
			logFatal("Can't specify OUTPUT file when generating backups")
		}
		logInfo("PGP private key detected, generating backup codes")
		pgpBackup(input)

	case *packet.PublicKey:
		outputW := pickOutput()
		if pgpArmor {
			outputW, err = armor.Encode(outputW, "PGP PRIVATE KEY BLOCK", nil)
			fatalIfErr(err)
		}
		logInfo("PGP public key detected, regenerating private key")
		pgpRestore(input, outputW)

	default:
		logFatal("Unrecognized PGP packet: %T", p)
	}

	return true
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

func pgpBackup(input []byte) {
	var passphrase = []byte("")
	var numWords int
	r := packet.NewReader(bytes.NewReader(input))
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
	fmt.Fprint(os.Stderr, "\n")
	logInfo("You will be able to regenerate the secret key by running this")
	logInfo("tool again on the public key and typing the provided %d words", numWords)
	logInfo("Testing the restore process is highly recommended")
}

func pgpRestore(input []byte, outputW io.WriteCloser) {
	r := packet.NewOpaqueReader(bytes.NewReader(input))
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
	outputW.Close()
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
