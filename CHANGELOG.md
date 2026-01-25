# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.2] - 2026-01-25

### Added
- `login`, `logout`, `status` commands for CLI authentication (OAuth device flow)
- `scan today` command to filter scans by current date
- API GET methods for server-mode scan queries (`GetScans`, `GetScan`)
- `--keep-local` flag for sync command to preserve local files
- Source-aware scan commands (API vs local based on server.enabled)
- Token management in `internal/auth/token.go`

### Changed
- `scan list` now queries API when server mode enabled
- `scan show` now queries API when server mode enabled
- `sync now` deletes local files after successful sync (unless `--keep-local`)
- Simplified auth commands to top-level (no `auth` subcommand)

### Fixed
- Scans no longer persist locally after successful server sync
- Source of truth now correctly follows server.enabled configuration

## [0.1.1] - 2026-01-24

### Changed
- Simplified README.md to match CLI implementation

### Removed
- SECURITY.md

### Fixed
- Hook command now accepts --event flag for proper event categorization
- Fixed redundant code and error handling in Cursor hook installation
- Fixed incomplete path traversal validation in scan loading
- Updated model pricing to use prefix matching for accurate cost estimates

## [0.1.0] - 2026-01-18

### Added
- Initial release
- Hook management for Cursor, Claude Code, and Gemini CLI
- Event normalization across AI tools
- Scan aggregation
- Local storage with optional server sync
- HMAC authentication for server sync

[0.1.2]: https://github.com/atbabers/intentra-cli/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/atbabers/intentra-cli/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/atbabers/intentra-cli/releases/tag/v0.1.0
