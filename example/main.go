// Command example shows how a program loads its configuration with envpgp.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/NoxPay/envPGP"
)

// Config mirrors the JSON document that was encrypted into env.pgp.
type Config struct {
	DatabaseURL string `json:"DATABASE_URL"`
	APIKey      string `json:"API_KEY"`
	JWTKey      string `json:"JWT_KEY"`
}

func usage() {
	fmt.Print(`usage: example [-env file] [-key file]

Asks for the passphrase of the private key, decrypts the configuration and
reports which fields were loaded (values are never printed).

flags:
  -env file   PGP-encrypted JSON configuration (default "env.pgp")
  -key file   passphrase-protected private key (default "private_key.pgp")

example:
  go run ./example -env env.pgp -key private_key.pgp
`)
}

func main() {
	envFile := flag.String("env", "env.pgp", "")
	keyFile := flag.String("key", "private_key.pgp", "")
	flag.Usage = usage
	flag.Parse()

	passphrase, err := envpgp.ReadPassphrase("Passphrase: ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var cfg Config
	err = envpgp.Load(&cfg, *envFile, *keyFile, passphrase)
	clear(passphrase)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("DATABASE_URL: %d bytes\n", len(cfg.DatabaseURL))
	fmt.Printf("API_KEY:      %d bytes\n", len(cfg.APIKey))
	fmt.Printf("JWT_KEY:      %d bytes\n", len(cfg.JWTKey))
}
