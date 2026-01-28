# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-01-26

### Added
- `intentra install [tool]` as top-level command (replaces `intentra hooks install`)
- `intentra uninstall [tool]` as top-level command (replaces `intentra hooks uninstall`)
- GitHub Copilot hook support (normalizer + hook installation)
- Windsurf Cascade hook support (normalizer + hook installation)
- `NormalizedType` field on Event struct for unified event classification across tools
- `RawEvents` field on Scan struct for forwarding raw hook data to backend
- `make intentra` command to build and install CLI to ~/bin/intentra
- Automatic token refresh using refresh tokens (sessions no longer expire after 24 hours)
- Server-side token/cost estimation for Claude Code and Gemini CLI hooks (tools that don't provide usage data)
- Scans now include conversation_id and session_id for traceability across tools
- Gemini CLI added to tools dropdown in frontend
- **Event aggregation**: Hook events are now buffered and sent as a single scan on "stop" event, enabling proper violation detection (retry loops, tool misuse, etc.)

### Changed
- **Normalizer architecture refactored**: Simplified interface with auto-registration pattern; normalizers only convert event types, raw data forwarded to backend
- `intentra login` now fails with error if already logged in (must logout first)
- Increased device registration timeout from 10s to 60s to handle Lambda cold starts
- All commands now suppress usage/help output on errors for cleaner CLI experience
- Scans page now filters out $0 cost scans by default (can be toggled)
- Scan detail page shows conversation_id (Cursor) or session_id (Claude/Gemini)
- Hook events buffered in temp directory (auto-cleared after 30 minutes or on successful send) - no permanent local storage for server-sync mode
- Scan API payload simplified to single-scan format with flat structure (was batch with nested `scans` array)
- Hook handler no longer requires server validation to start (silent no-op when server disabled)

### Removed
- `intentra hooks install` and `intentra hooks uninstall` commands (use `intentra install` and `intentra uninstall`)
- SQLite-based offline buffer (`internal/buffer`) - server-sync mode now uses temp file buffering only
- Violation model (`pkg/models/violation.go`) - violation detection moved to backend
- `NormalizedEvent` struct - raw events forwarded directly to backend for processing
- `Retries` field from Scan struct - retry detection moved to backend
- Hardcoded `HookType` constants - native event types preserved, normalized separately

### Fixed
- Hooks now send scans using JWT auth (was incorrectly using API key headers)
- Scan submission now accepts HTTP 201 Created response
- Scan submission payload now matches backend schema (was sending nested `scans` array)
- Hook data now properly normalized from Cursor/Claude Code/Gemini CLI (maps field names like `hook_event_name`→`hook_type`, `tool_response`→`tool_output`, `duration`→`duration_ms`, extracts `command` from `tool_input`)
- Raw hook events now forwarded to backend for violation detection (all three tools provide rich data: `tool_name`, `tool_input`, `tool_output`/`tool_response`, `duration`, `session_id`/`conversation_id`)
- Backend violation detector now recognizes `hook_type` field from CLI events
- Single-event retry detection for obvious patterns ("retrying", "connection refused", etc.)
- Machines Lambda now has access to orgs table for plan resolution
- Version now correctly reported in device metadata (was always showing "dev")
- Added ldflags for `internal/device.Version` in Makefile and GoReleaser config
- Status message now shows "Not logged in" instead of "Not authenticated"
- Help text now correctly references `intentra login` instead of `intentra auth login`
- Backend token endpoint now returns OAuth2-compliant error format for device flow polling

## [0.1.3] - 2026-01-25

### Added
- Auto-register device on login via `POST /machines`
- Handle device limit errors with upgrade prompt
- Handle admin-revoked device errors with support message

### Changed
- Simplified fallback device ID to `hostname:username` (removed home directory component)

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
- Updated README to emphasize local-first mode with optional intentra.sh server sync

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

[0.2.0]: https://github.com/atbabers/intentra-cli/compare/v0.1.3...v0.2.0
[0.1.3]: https://github.com/atbabers/intentra-cli/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/atbabers/intentra-cli/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/atbabers/intentra-cli/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/atbabers/intentra-cli/releases/tag/v0.1.0
