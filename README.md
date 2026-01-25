# Intentra CLI

Open-source monitoring tool for AI coding assistants. Captures events from Cursor, Claude Code, and Gemini CLI, normalizes them into a unified schema, and aggregates them into scans.

## Installation

```bash
curl -fsSL https://install.intentra.sh | sh
```

Or with Homebrew:

```bash
brew install atbabers/intentra/intentra
```

Verify installation:

```bash
intentra --version
```

## Quick Start

```bash
intentra hooks install
intentra hooks status
intentra scan list
```

## Commands

| Command | Description |
|---------|-------------|
| `intentra hooks install` | Install hooks for AI tools |
| `intentra hooks uninstall` | Remove hooks from AI tools |
| `intentra hooks status` | Check hook installation status |
| `intentra scan list` | List captured scans |
| `intentra scan show <id>` | Show scan details |
| `intentra scan aggregate` | Process events into scans |
| `intentra sync now` | Sync scans to server |
| `intentra sync status` | Show sync status |
| `intentra config show` | Display configuration |
| `intentra config init` | Generate sample config |
| `intentra config validate` | Validate configuration |

## Supported Tools

| Tool | Status |
|------|--------|
| Cursor | Supported |
| Claude Code | Supported |
| Gemini CLI | Supported |

## Configuration

Configuration file location: `~/.config/intentra/config.yaml`

```yaml
server:
  enabled: true
  endpoint: "https://api.intentra.sh/v1"
  auth:
    mode: "hmac"
    hmac:
      key_id: "your-key-id"
      secret: "your-secret"
```

## Documentation

Full documentation: [docs.intentra.sh](https://docs.intentra.sh)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[Apache License 2.0](LICENSE)
