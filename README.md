# envPGP

Load program configuration from a JSON document encrypted with PGP.

The server keeps two files: `env.pgp` (the encrypted JSON) and the private key,
itself protected by a passphrase. Neither is of any use without the passphrase,
which is typed when the program starts. The decrypted JSON only ever exists in
the memory of the process.

## Usage

```go
type Config struct {
	DatabaseURL string `json:"DATABASE_URL"`
	APIKey      string `json:"API_KEY"`
}

passphrase, err := envpgp.ReadPassphrase("Passphrase: ")
if err != nil {
	return err
}

var cfg Config
err = envpgp.Load(&cfg, "env.pgp", "private_key.pgp", passphrase)
clear(passphrase)
```

`ReadPassphrase` reads from the terminal without echo; when stdin is not a
terminal it reads one line, so the passphrase can be piped in. `Decrypt` is the
same operation on byte slices, for callers that do not keep the files on disk.

A private key stored without a passphrase is refused (`ErrUnprotectedKey`).

A runnable program is in [example](example/main.go):

```bash
go run ./example
```

## Creating env.pgp

```bash
gpg --import public_key.pgp
```

```bash
gpg --armor --encrypt --recipient <key id> --output env.pgp env.json
```

Only the public key is needed to encrypt, so the plaintext `env.json` never has
to be on the server.
