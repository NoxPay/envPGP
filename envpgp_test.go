package envpgp

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

const testPassphrase = "correct horse"

type testConfig struct {
	DatabaseURL string `json:"DATABASE_URL"`
	APIKey      string `json:"API_KEY"`
}

func armored(t *testing.T, blockType string, write func(w *bytes.Buffer) error) []byte {
	t.Helper()

	var raw bytes.Buffer
	err := write(&raw)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	w, err := armor.Encode(&out, blockType, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write(raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}

	return out.Bytes()
}

// fixture returns an armored private key (protected when protect is true) and
// an armored message carrying plaintext encrypted to it.
func fixture(t *testing.T, plaintext string, protect bool) (key, message []byte) {
	t.Helper()

	e, err := openpgp.NewEntity("test", "", "test@example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	message = armored(t, "PGP MESSAGE", func(buf *bytes.Buffer) error {
		w, err := openpgp.Encrypt(buf, []*openpgp.Entity{e}, nil, nil, nil)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(plaintext))
		if err != nil {
			return err
		}
		return w.Close()
	})

	if protect {
		err = e.EncryptPrivateKeys([]byte(testPassphrase), nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	key = armored(t, openpgp.PrivateKeyType, func(buf *bytes.Buffer) error {
		return e.SerializePrivateWithoutSigning(buf, nil)
	})

	return key, message
}

func TestLoad(t *testing.T) {
	key, message := fixture(t, `{"DATABASE_URL":"postgres://x","API_KEY":"k"}`, true)

	dir := t.TempDir()
	envFile := filepath.Join(dir, "env.pgp")
	keyFile := filepath.Join(dir, "private_key.pgp")
	err := os.WriteFile(envFile, message, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(keyFile, key, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	var cfg testConfig
	err = Load(&cfg, envFile, keyFile, []byte(testPassphrase))
	if err != nil {
		t.Fatal(err)
	}

	want := testConfig{DatabaseURL: "postgres://x", APIKey: "k"}
	if cfg != want {
		t.Fatalf("got %+v, want %+v", cfg, want)
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	key, message := fixture(t, `{}`, true)

	_, err := Decrypt(message, key, []byte("wrong"))
	if err == nil {
		t.Fatal("expected an error for a wrong passphrase")
	}
}

func TestDecryptUnprotectedKey(t *testing.T) {
	key, message := fixture(t, `{}`, false)

	_, err := Decrypt(message, key, []byte(testPassphrase))
	if !errors.Is(err, ErrUnprotectedKey) {
		t.Fatalf("got %v, want ErrUnprotectedKey", err)
	}
}

func TestDecryptOtherKey(t *testing.T) {
	_, message := fixture(t, `{}`, true)
	key, _ := fixture(t, `{}`, true)

	_, err := Decrypt(message, key, []byte(testPassphrase))
	if err == nil {
		t.Fatal("expected an error for a message encrypted to another key")
	}
}
