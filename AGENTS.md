# AGENTS.md

Guidance for AI agents that use this package in another program, or that
change this repository.

## What it is

`envpgp` loads a program's configuration from a JSON document encrypted with
PGP. The private key is itself protected by a passphrase that a person types
when the program starts, so the files on the server are useless on their own.
The decrypted JSON only exists in process memory.

Use it when a service's secrets must not sit readable on disk or in the
environment. Do not use it for a program that has to restart unattended: there
is nobody to type the passphrase.

## Using it in a program

The module path and the package name differ:

```go
import "github.com/NoxPay/envPGP" // package envpgp
```

The repository is private; the consumer needs
`go env -w GOPRIVATE=github.com/NoxPay/*`.

Call it once, at the top of `main`, before anything that needs configuration:

```go
type Config struct {
	DatabaseURL string `json:"DATABASE_URL"`
	APIKey      string `json:"API_KEY"`
}

func main() {
	passphrase, err := envpgp.ReadPassphrase("Passphrase: ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var cfg Config
	err = envpgp.Load(&cfg, "env.pgp", "private_key.pgp", passphrase)
	clear(passphrase)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// cfg is ready.
}
```

The whole API:

| Function | Use |
|---|---|
| `ReadPassphrase(prompt)` | Reads the passphrase from the terminal without echo. When stdin is not a terminal it reads one line, so tests and scripts can pipe it in. The prompt goes to stderr. |
| `Load(v, envFile, keyFile, passphrase)` | Reads both files, decrypts, and unmarshals the JSON into `v` (a pointer to a struct with `json` tags). |
| `Decrypt(message, privateKey, passphrase)` | The same on byte slices, returning the plaintext. For callers that do not keep the files on disk. |
| `ErrUnprotectedKey` | Returned when the private key has no passphrase. Check with `errors.Is`. |

A complete program is in [example/main.go](example/main.go).

## Rules when writing code that uses it

- The passphrase comes from `ReadPassphrase`. Never read it from a flag, an
  environment variable, a file, or a constant: each of those puts it back on
  the machine, which is what the package exists to avoid.
- `clear(passphrase)` right after `Load` returns, on the error path too.
- Never log or print the loaded struct or any of its fields. The example
  prints them only because its values are placeholders.
- Do not work around `ErrUnprotectedKey` by decrypting the key some other way.
  Protect the key with a passphrase instead.
- Both files must be ASCII-armored. A binary `.gpg` file is rejected; re-export
  with `gpg --armor` rather than adding a conversion step.
- Unknown JSON keys are ignored and missing ones leave the field at its zero
  value, so validate required fields after `Load`.
- A wrong passphrase surfaces as an "unlock private key" error. Report it and
  exit; do not loop asking again unless the user asked for that behavior.
- `Load` holds neither file open. The operator may delete `env.pgp` and the
  private key once the program is up, so never re-read them later (no reload
  on signal, no lazy loading).

## Files that never go into a repository

`env.json`, `env.pgp` and any `*.pgp` key are ignored by this repository's
`.gitignore`. Keep the same entries in a consuming repository. To create them
for local work, follow "Running the example" in [README.md](README.md); do not
generate keys or passphrases on the user's behalf without being asked, and
never put a real passphrase in a file, a command you save, or a commit.

## Changing this repository

- Packages live at the repository root; the example is in `example/`. No
  `internal/`.
- Tests generate their keys in memory. Do not add key or message fixtures to
  the tree.
- Two direct dependencies, each for something the standard library lacks:
  `github.com/ProtonMail/go-crypto` (OpenPGP) and `golang.org/x/term`
  (reading without echo). Adding a third needs a stated reason.
- Inside the NoxPay workspace the parent `go.work` does not list this module,
  so run Go commands with `GOWORK=off`.
- Before calling a change done: `go fix ./...`, `modernize -fix ./...`,
  `gofmt -l .`, `go vet ./...`, `golangci-lint run ./...`, `gosec ./...`,
  `staticcheck ./...`, `go test -timeout 30s -count 1 ./...`, and
  `golangci-lint` again with `GOOS=linux` and `GOOS=windows`. CI
  (`.github/workflows/ci.yml`) runs the same checks on linux, macOS and
  windows.
- A change to the API updates this file, `README.md` and `example/main.go`
  together.
