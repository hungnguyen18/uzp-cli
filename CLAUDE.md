# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

UZP-CLI (User's Zecure Pocket) is a Go CLI tool for securely storing sensitive data locally (API keys, tokens, credentials). All data is encrypted with AES-256-GCM + scrypt key derivation and stored in `~/.uzp/uzp.vault`.

## Build & Development Commands

```bash
# Quick dev build (outputs ./uzp binary)
npm run build-dev

# Cross-platform release build (linux/darwin/windows, amd64/arm64)
npm run build

# Run tests
go test ./...
go test -v -race -coverprofile=coverage.out ./...

# Lint
golangci-lint run

# Security check
govulncheck ./...

# Manual testing
./uzp init          # Create vault with master password
./uzp add           # Add a secret (interactive)
./uzp get <project>/<key>
./uzp list
```

## Architecture

**Go CLI built with Cobra.** Entry: `main.go` -> `cmd.Execute()`.

```
cmd/           CLI commands (one file per command, registered in init() via cobra)
  root.go      Root command, version flag, global vault instance
  helpers.go   ensureVaultUnlocked() - shared password prompt logic
  init|add|get|copy|update|list|search|inject|reset.go

internal/
  crypto/      AES-256-GCM encryption, scrypt key derivation, password hashing
  storage/     Vault struct: load/save encrypted JSON, CRUD operations on secrets
  utils/       Clipboard with auto-clear TTL (default 15s)
```

**Data flow:** User command -> `ensureVaultUnlocked()` prompts password -> `storage.Vault` decrypts `~/.uzp/uzp.vault` -> operation -> re-encrypt and save.

**Vault file format:** JSON with salt (32 bytes), password hash (scrypt), and AES-256-GCM encrypted data blob. File permissions: 0600.

## Key Design Decisions

- Stateless per invocation: master password prompted every time (never cached)
- Secrets organized by project/key pairs
- `inject` command exports secrets as `.env` format for shell usage
- `copy` command uses clipboard with configurable TTL auto-clear
- Sensitive data cleared from memory after use (`defer` zero-fill patterns)

## Git & Release

- Branch naming: `feature/`, `bug/`, `hotfix/`, `docs/`, `security/`, `test/`, `refactor/`, `perf/`, `devops/`, `misc/` + `snake_case`
- Commit format: `feat:`, `fix:`, `docs:`, `security:`, `test:`, `refactor:`, `perf:`, `hotfix:`, `ci:`, `misc:`
- Release: `npm run release <version>` or `./scripts/release.sh <version>` (auto-builds, tags, publishes to npm + GitHub)
- Version injected via `-ldflags` from `package.json` version field

## CI/CD Pipeline

Three workflows in `.github/workflows/`:

- **ci.yml** — Runs on PR/push to main: `go test -race` (Go 1.21/1.22/1.23.10 matrix), `golangci-lint`, `govulncheck`, cross-platform builds, npm package validation
- **auto-release.yml** — Triggered by `v*` tags: builds binaries, creates GitHub release, calls publish-package
- **publish-package.yml** — Publishes to npm and GitHub Packages with idempotency checks and retry logic

Security-critical files (`internal/crypto/`, `internal/storage/`, `.github/workflows/`, `go.mod`) require owner review via CODEOWNERS.

## npm Distribution

The npm package is a binary wrapper. `scripts/install.js` downloads the correct platform binary on `npm install`. The Go binary is the actual tool.
