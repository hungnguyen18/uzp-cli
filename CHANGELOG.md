# Changelog

## [v1.2.0](https://github.com/hungnguyen18/uzp-cli/releases/tag/v1.2.0) — 2026-03-29

### Added
- feat: add template, diff, audit, backup/restore, rotate commands

### Fixed
- fix: handle errcheck lint for MCP prompt, read from /dev/tty instead of stderr

### Other
- docs: update README with v1.1.0 features (run, import, mcp) and changelog section
- docs: update CHANGELOG.md for v1.1.0

## [v1.1.0](https://github.com/hungnguyen18/uzp-cli/releases/tag/v1.1.0) — 2026-03-28

### Added
- feat: add uzp mcp command for AI agent integration
- feat: add MCP tool handlers (get, list, search) with access control
- feat: add MCP JSON-RPC server with stdio transport
- feat: add MCP access control with glob pattern matching
- feat: add uzp import command for .env file migration
- feat: add .env file parser with quoted value support
- feat: add uzp run command for secret injection into process environment
- feat: add MergeSecrets to envutil for multi-project environment building

### Other
- chore: bump version to 1.1.0
- docs: add uzp mcp to root help text
- docs: add uzp import to root help text
- docs: add uzp run to root help text
- test: add validation tests for uzp run command
- refactor: extract ConvertToEnvKey into shared envutil package
- docs: add implementation plans for uzp v2 (run, import, mcp)
- docs: add UZP v2 feature design spec (run, import, mcp)
- docs: update CHANGELOG.md for v1.0.16

## [v1.0.16](https://github.com/hungnguyen18/uzp-cli/releases/tag/v1.0.16) — 2026-03-28

### Security
- security: fix 11 vulnerabilities across crypto, storage, and CLI

### Added
- feat: auto-update CHANGELOG.md on release

### Other
- chore: bump version to 1.0.16
- Fix Go version to 1.24 for golangci-lint compatibility
- Update Go to 1.25 and upgrade all dependencies
- Fix build: update HashPassword call signature and upgrade CI actions
- Remove CI/CD dead code and rewrite CONTRIBUTING.md
- Create CODE_OF_CONDUCT.md
- Potential fix for code scanning alert no. 20: Use of a broken or weak cryptographic hashing algorithm on sensitive data
