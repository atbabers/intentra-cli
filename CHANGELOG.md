# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial open source release
- Hook support for Cursor, Claude Code, and Gemini
- Event normalization across different AI tool formats
- Scan aggregation (grouping events into prompt-completion cycles)
- Local-first architecture with optional server sync
- HMAC, API key, and mTLS authentication modes
- Configuration via file, environment variables, and CLI flags

### Changed
- N/A

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- N/A

### Security
- N/A

## [0.1.0] - 2026-01-18

### Added
- Initial release
- `intentra hooks install` - Install monitoring hooks
- `intentra hooks uninstall` - Remove monitoring hooks
- `intentra hooks status` - Check hook installation status
- `intentra scan list` - List captured scans
- `intentra scan show` - View scan details
- `intentra scan aggregate` - Process events into scans
- `intentra sync now` - Sync scans to server
- `intentra sync status` - View sync status
- `intentra config show` - Display configuration
- `intentra config init` - Generate sample config
- `intentra config validate` - Validate configuration

[Unreleased]: https://github.com/atbabers/intentra-cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/atbabers/intentra-cli/releases/tag/v0.1.0
