package main

import (
	cryptoRand "crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/cloudflare/circl/dh/x25519"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	keyPathFlag = flag.String("key-path", "", "")
	cmdFlag     = flag.String("cmd", "keygen", "operations are either 'keygen' to make a new key or 'print-pubkey' to print the hex-encoded public key of a private key")
)

func main() {
	if err := mainErr(); err != nil {
		logrus.Fatalf(err.Error())
	}
}

func mainErr() error {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Commmands:
* -cmd=keygen -key-path=blah
  
    Generate a new private key and store it in the file 'blah'

* -cmd=print-pubkey -key-path=blah
  
    Print the hex-encoded public key of the private key stored in the file 'blah'
`)
		flag.PrintDefaults()
	}
	flag.Parse()
	if *keyPathFlag == "" {
		return errors.New("key-path is empty")
	}
	switch *cmdFlag {
	case "keygen":
		var privKey, pubKey x25519.Key
		_, err := io.ReadFull(cryptoRand.Reader, privKey[:])
		if err != nil {
			return errors.Wrap(err, "")
		}
		x25519.KeyGen(&pubKey, &privKey)
		err = os.WriteFile(*keyPathFlag, privKey[:], 0o600)
		if err != nil {
			return errors.Wrap(err, "")
		}
		logrus.Infof("Private key written in %s", *keyPathFlag)
		logrus.Infof("Hex-encoded public key is %s", hex.EncodeToString(pubKey[:]))
	case "print-pubkey":
		b, err := os.ReadFile(*keyPathFlag)
		if err != nil {
			return errors.Wrap(err, "")
		}
		var privKey, pubKey x25519.Key
		copy(privKey[:], b)
		x25519.KeyGen(&pubKey, &privKey)
		logrus.Infof("Hex-encoded public key is %s", hex.EncodeToString(pubKey[:]))
	default:
		return errors.New("Unknown cmd")
	}

	return nil
}
