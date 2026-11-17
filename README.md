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

## Running the example

The repository carries no key and no configuration file: `env.json` and
`*.pgp` are in `.gitignore`. Create them in the repository root with GnuPG.

1. Generate a key pair. GnuPG asks for the passphrase that will protect the
   private key; it is the one the program asks for at startup.

```bash
gpg --quick-generate-key "envPGP example <envpgp@example.com>" default default never
```

2. Export both halves. The exported private key stays protected by the
   passphrase.

```bash
gpg --armor --export envpgp@example.com > public_key.pgp
```

```bash
gpg --armor --export-secret-keys envpgp@example.com > private_key.pgp
```

3. Write `env.json`. The [example](example/main.go) expects this structure:

```json
{
  "DATABASE_URL": "postgres://...",
  "API_KEY": "...",
  "JWT_KEY": "..."
}
```

4. Encrypt it to the public key.

```bash
gpg --armor --encrypt --recipient envpgp@example.com --output env.pgp env.json
```

5. Run the example and type the passphrase.

```bash
go run ./example
```

```
Passphrase:
DATABASE_URL: postgres://...
API_KEY:      ...
JWT_KEY:      ...
```

The example prints the values because they are placeholders; a real program
should not.

## On a server

The program needs only `env.pgp` and `private_key.pgp`. Encrypting takes just
the public key, so the plaintext `env.json` never has to be on the server, and
`public_key.pgp` is all a colleague needs to produce a new `env.pgp`.

Both files must be ASCII-armored (`--armor`).

### Removing the files after startup

`Load` reads both files once and holds nothing open. Once the program is up,
`env.pgp` and `private_key.pgp` can be deleted from the server and kept only
by the operator: from then on the configuration exists nowhere on the machine
but in the memory of the process.

The cost is that a restart is no longer unattended. Someone has to copy the
two files back and type the passphrase again, so this suits long-running
services, not ones that are restarted by a supervisor.
