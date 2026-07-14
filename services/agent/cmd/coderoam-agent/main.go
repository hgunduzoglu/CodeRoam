package main

import (
	"crypto/rand"
	"encoding/base32"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	flag.Parse()
	command := "run"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}

	switch command {
	case "run":
		log.Print("CodeRoam agent starter: outbound relay session not implemented")
	case "pair":
		secret, err := pairingSecret()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("CodeRoam pairing secret (starter placeholder):")
		fmt.Println(secret)
		fmt.Println("The production flow encodes this in a QR payload and uses Noise XXpsk3.")
	case "version":
		fmt.Println("coderoam-agent 0.1.0-dev")
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command)
		os.Exit(2)
	}
}

func pairingSecret() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate pairing secret: %w", err)
	}
	return strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buffer), "="), nil
}
