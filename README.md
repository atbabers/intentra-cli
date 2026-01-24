<p align="center">
  <h1 align="center">Intentra CLI</h1>
  <p align="center">
    <strong>Monitor and audit AI coding assistants for policy compliance</strong>
  </p>
</p>

<p align="center">
  <a href="https://github.com/atbabers/intentra-cli/actions/workflows/ci.yml"><img src="https://github.com/atbabers/intentra-cli/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/atbabers/intentra-cli/releases"><img src="https://img.shields.io/github/v/release/atbabers/intentra-cli" alt="Release"></a>
  <a href="https://goreportcard.com/report/github.com/atbabers/intentra-cli"><img src="https://goreportcard.com/badge/github.com/atbabers/intentra-cli" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License"></a>
  <a href="https://pkg.go.dev/github.com/atbabers/intentra-cli"><img src="https://pkg.go.dev/badge/github.com/atbabers/intentra-cli.svg" alt="Go Reference"></a>
</p>

---

Intentra CLI is an open-source tool that monitors AI coding assistants (Cursor, Claude Code, Gemini) for usage tracking, policy compliance, and security auditing. It installs lightweight hooks into your AI tools to capture and normalize events, aggregates them into scans, and optionally syncs data to a central server.

## Features

- **Multi-tool Support** - Works with Cursor, Claude Code, and Gemini
- **Unified Schema** - Normalizes events across different AI tools into a consistent format
- **Scan Aggregation** - Groups related events into logical "scans" (prompt-to-completion cycles)
- **Local-first** - Works entirely offline; server sync is optional
- **Flexible Auth** - Supports API key, HMAC, and mTLS authentication modes
- **Privacy-focused** - You control what data is collected and where it goes

## Installation

### Using Go

```bash
go install github.com/atbabers/intentra-cli/cmd/intentra@latest
```

### From Source

```bash
git clone https://github.com/atbabers/intentra-cli.git
cd intentra-cli
make build
```

### Pre-built Binaries

Download pre-built binaries from the [Releases](https://github.com/atbabers/intentra-cli/releases) page.

## Quick Start

### 1. Install Hooks

Install monitoring hooks for all supported AI tools:

```bash
intentra hooks install
```

Or install for a specific tool:

```bash
intentra hooks install --tool cursor
intentra hooks install --tool claude
```

### 2. Check Installation Status

```bash
intentra hooks status
```

### 3. View Captured Data

After using your AI tools, aggregate events into scans:

```bash
intentra scan aggregate
intentra scan list
```

### 4. (Optional) Configure Server Sync

Create a configuration file at `~/.config/intentra/config.yaml`:

```yaml
server:
  enabled: true
  endpoint: "https://your-server.com/api/v1"
  timeout: 30s
  auth:
    mode: "hmac"
    hmac:
      key_id: "your-key-id"
      secret: "your-secret"
```

Then sync your scans:

```bash
intentra sync now
```

## Commands

| Command | Description |
|---------|-------------|
| `intentra hooks install` | Install hooks for AI tools |
| `intentra hooks uninstall` | Remove hooks from AI tools |
| `intentra hooks status` | Check hook installation status |
| `intentra scan list` | List all captured scans |
| `intentra scan show <id>` | Show details of a specific scan |
| `intentra scan aggregate` | Process events into scans |
| `intentra sync now` | Sync pending scans to server |
| `intentra sync status` | Show sync status |
| `intentra config show` | Display current configuration |
| `intentra config init` | Generate sample configuration |
| `intentra config validate` | Validate configuration |

## Configuration

Intentra looks for configuration in the following locations (in order):

1. Path specified via `--config` flag
2. `~/.config/intentra/config.yaml`
3. Environment variables (prefixed with `INTENTRA_`)

### Configuration Options

```yaml
server:
  enabled: true
  endpoint: "https://api.example.com/v1"
  timeout: 30s
  auth:
    mode: "hmac"  # Options: api_key, hmac, mtls
    api_key:
      key_id: ""
      secret: ""
    hmac:
      key_id: ""
      secret: ""
      device_id: ""
    mtls:
      cert_path: ""
      key_path: ""
      ca_path: ""

buffer:
  path: "~/.local/share/intentra/buffer"
  max_size: 1000
  flush_interval: 60s
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `INTENTRA_SERVER_ENDPOINT` | API server endpoint |
| `INTENTRA_SERVER_AUTH_MODE` | Authentication mode |
| `INTENTRA_API_KEY_ID` | API key ID |
| `INTENTRA_API_SECRET` | API secret |

## Supported AI Tools

| Tool | Hook Location | Status |
|------|---------------|--------|
| Cursor | `~/.cursor/` | ✅ Supported |
| Claude Code | `~/.claude/` | ✅ Supported |
| Gemini | Platform-specific | ✅ Supported |

## Data Flow

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐     ┌────────────┐
│  AI Tool    │────▶│   Intentra   │────▶│   Buffer    │────▶│   Server   │
│  (Cursor)   │     │    Hook      │     │   (Local)   │     │ (Optional) │
└─────────────┘     └──────────────┘     └─────────────┘     └────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  Normalize   │
                    │   Schema     │
                    └──────────────┘
```

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for convenience)

### Building

```bash
make build
```

### Running Tests

```bash
make test
```

### Linting

```bash
make lint
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## Security

For security issues, please see our [Security Policy](SECURITY.md).

## License

Intentra CLI is licensed under the [Apache License 2.0](LICENSE).

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
