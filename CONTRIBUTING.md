# Contributing to UZP-CLI

Thank you for your interest in contributing. This guide covers everything you need to get started.

## Quick Start

```bash
# Fork on GitHub, then clone
git clone https://github.com/YOUR_USERNAME/uzp-cli.git
cd uzp-cli

# Install dependencies
go mod download && npm install

# Build and verify
go build -o uzp . && ./uzp --help

# Create your branch
git checkout -b feature/your_feature_name
```

## Prerequisites

- **Go** 1.23.10+
- **Node.js** 18+
- **Git** 2.30+

## Development

### Build

```bash
npm run build-dev          # Quick local build (outputs ./uzp)
npm run build              # Cross-platform build (all targets)
```

### Test

```bash
go test ./...              # Run all tests
go test -v -race ./...     # Verbose with race detection

# Manual verification
./uzp init
./uzp add
./uzp get <project>/<key>
```

### Lint

```bash
golangci-lint run
```

## Branch Naming

Format: `<type>/<description_in_snake_case>`

| Prefix      | Use for                  | Commit prefix |
| ----------- | ------------------------ | ------------- |
| `feature/`  | New functionality        | `feat:`       |
| `bug/`      | Bug fixes                | `fix:`        |
| `hotfix/`   | Urgent production fixes  | `hotfix:`     |
| `security/` | Security improvements    | `security:`   |
| `docs/`     | Documentation            | `docs:`       |
| `test/`     | Tests                    | `test:`       |
| `refactor/` | Code restructuring       | `refactor:`   |
| `perf/`     | Performance              | `perf:`       |
| `devops/`   | CI/CD and infrastructure | `ci:`         |
| `misc/`     | Minor cleanups           | `misc:`       |

## Commit Messages

```
<type>: <description>
```

Examples:

```
feat: add vault export functionality
fix: prevent clipboard memory leak
security: strengthen password validation
```

## Code Style

### Go

- Use clear, descriptive names for functions and variables.
- Always handle errors explicitly with `fmt.Errorf("context: %w", err)`.
- Clear sensitive data from memory after use:

```go
password := getPassword()
defer func() {
    for i := range password {
        password[i] = 0
    }
}()
```

- Validate all user inputs (length, format, path traversal).
- Use secure file permissions (`0600`) for vault files.
- Use `crypto/rand` for random generation, never `math/rand`.

## Review Process

Most PRs are reviewed and merged once CI passes. Only changes to security-critical paths require owner review:

- `internal/crypto/` and `internal/storage/`
- `.github/workflows/` and `go.mod`

These paths are enforced via [CODEOWNERS](.github/CODEOWNERS).

## Release Process (Maintainers)

```bash
./scripts/release.sh 1.0.8    # or: npm run release 1.0.8
```

This updates `package.json`, commits, tags, and pushes. GitHub Actions then builds cross-platform binaries, creates a GitHub release, and publishes to npm.

### Versioning

- `vX.Y.Z` where: patch = bug fixes, minor = new features, major = breaking changes
- Only `hungnguyen18` can trigger releases

## Project Structure

```
cmd/            CLI commands (Cobra)
internal/
  crypto/       AES-256-GCM encryption, scrypt key derivation
  storage/      Vault file I/O, secret CRUD operations
  utils/        Clipboard with auto-clear TTL
scripts/        Build, install, release automation
```

## Getting Help

- Search [existing issues](https://github.com/hungnguyen18/uzp-cli/issues) before creating new ones.
- Use [GitHub Discussions](https://github.com/hungnguyen18/uzp-cli/discussions) for questions.
- Report security vulnerabilities privately to the maintainer (see [SECURITY.md](SECURITY.md)).
