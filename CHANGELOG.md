# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[0.1.1]: https://github.com/atbabers/intentra-cli/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/atbabers/intentra-cli/releases/tag/v0.1.0
