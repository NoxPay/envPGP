// Package envpgp loads program configuration from a JSON document encrypted
// with PGP. The private key stays on disk protected by a passphrase, so the
// configuration can only be read by someone who types it at startup.
package envpgp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"golang.org/x/term"
)

// ErrUnprotectedKey is returned for a private key stored without a passphrase.
// Such a key would let anyone with the two files read the configuration, which
// is the one thing this package exists to prevent.
var ErrUnprotectedKey = errors.New("envpgp: private key is not protected by a passphrase")

// Load reads envFile and keyFile once and keeps neither open, so both can be
// removed from the machine as soon as it returns.
func Load(v any, envFile, keyFile string, passphrase []byte) error {
	message, err := os.ReadFile(envFile) // #nosec G304 -- path chosen by the caller
	if err != nil {
		return fmt.Errorf("envpgp: %w", err)
	}

	key, err := os.ReadFile(keyFile) // #nosec G304 -- path chosen by the caller
	if err != nil {
		return fmt.Errorf("envpgp: %w", err)
	}

	plain, err := Decrypt(message, key, passphrase)
	if err != nil {
		return err
	}
	defer clear(plain)

	// An editor may have saved env.json with a BOM, which encoding/json rejects.
	err = json.Unmarshal(bytes.TrimPrefix(plain, []byte("\xef\xbb\xbf")), v)
	if err != nil {
		return fmt.Errorf("envpgp: parse configuration: %w", err)
	}

	return nil
}

// Decrypt is Load without the files and the JSON step. Both inputs must be
// ASCII-armored.
func Decrypt(message, privateKey, passphrase []byte) ([]byte, error) {
	keyring, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(privateKey))
	if err != nil {
		return nil, fmt.Errorf("envpgp: read private key: %w", err)
	}

	for _, e := range keyring {
		if e.PrivateKey == nil {
			return nil, errors.New("envpgp: key has no private part")
		}
		if !e.PrivateKey.Encrypted {
			return nil, ErrUnprotectedKey
		}

		err = e.DecryptPrivateKeys(passphrase)
		if err != nil {
			return nil, fmt.Errorf("envpgp: unlock private key: %w", err)
		}
	}

	block, err := armor.Decode(bytes.NewReader(message))
	if err != nil {
		return nil, fmt.Errorf("envpgp: not an armored PGP message: %w", err)
	}

	md, err := openpgp.ReadMessage(block.Body, keyring, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("envpgp: decrypt: %w", err)
	}

	// The integrity tag is only checked once the body is read to EOF, so the
	// plaintext must not be used before ReadAll returns without error.
	plain, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return nil, fmt.Errorf("envpgp: read message body: %w", err)
	}

	return plain, nil
}

// ReadPassphrase writes prompt to stderr and reads a passphrase from stdin
// without echoing it. When stdin is not a terminal it reads one line instead,
// so the passphrase can be piped in.
func ReadPassphrase(prompt string) ([]byte, error) {
	fd := int(os.Stdin.Fd()) // #nosec G115 -- file descriptors fit in an int

	if !term.IsTerminal(fd) {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return nil, fmt.Errorf("envpgp: read passphrase: %w", err)
		}
		return []byte(strings.TrimRight(line, "\r\n")), nil
	}

	fmt.Fprint(os.Stderr, prompt)
	pass, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("envpgp: read passphrase: %w", err)
	}

	return pass, nil
}
