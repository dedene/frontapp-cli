# Repository Guidelines

## Project Structure

- `cmd/frontcli/`: CLI entrypoint
- `internal/`: implementation
  - `cmd/`: command routing (kong CLI framework, 27+ commands)
  - `api/`: Front API client
  - `auth/`: OAuth2 flow + keyring + certificate handling
  - `config/`: configuration + accounts + credentials
  - `markdown/`: HTML-to-Markdown converter
  - `ui/`: UI components
  - `output/`: table rendering + time formatting
  - `errfmt/`: error formatting
- `bin/`: build outputs

## Build, Test, and Development Commands

- `make build`: compile to `bin/frontcli`
- `make front -- <args>` or `make frontcli`: build + run
- `make fmt` / `make lint` / `make test` / `make ci`: format, lint, test, full local gate
- `make tools`: install pinned dev tools into `.tools/`
- `make clean`: remove bin/ and .tools/

## Coding Style & Naming Conventions

- Formatting: `make fmt` (goimports local prefix `github.com/dedene/frontapp-cli` + gofumpt)
- Output: keep stdout parseable (`--json`); send human hints/progress to stderr
- Linting: golangci-lint with project config

## Testing Guidelines

- Unit tests: stdlib `testing` (files: `*_test.go` next to code)
- Coverage areas: conversations, credentials, config, output (table/time), API (idprefix/client), errfmt
- 7 test files total

## Config & Secrets

- **OAuth2**: golang.org/x/oauth2 for authentication flow
- **Keyring**: 99designs/keyring for token storage
- **Cert handling**: custom certificate support for API connections
- **Credential caching**: tokens cached for performance
- **Multi-account**: supports multiple Front accounts

## Key Commands

- `conversations`: list/manage conversations
- `drafts`: handle draft messages
- `tags`, `teammates`, `templates`, `inboxes`, `messages`, `contacts`, `channels`: domain operations
- `auth`: authentication management

## Commit & Pull Request Guidelines

- Conventional Commits: `feat|fix|refactor|build|ci|chore|docs|style|perf|test`
- Group related changes; avoid bundling unrelated refactors
- PR review: use `gh pr view` / `gh pr diff`; don't switch branches

## Security Tips

- Never commit OAuth client credentials or tokens
- Prefer OS keychain backends for credential storage
- Certificate files should not be committed to repository
