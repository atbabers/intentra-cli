# Intentra CLI

Open-source monitoring tool for AI coding assistants. Captures events from Cursor, Claude Code, Gemini CLI, GitHub Copilot, and Windsurf, normalizes them into a unified schema, and aggregates them into scans.

**Local-first by default** - all data stays on your machine. For advanced observability and team features, connect to [intentra.sh](https://intentra.sh).

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
intentra install
intentra hooks status
intentra scan list
```

All scans are stored locally at `~/.local/share/intentra/scans/`.

## Commands

| Command | Description |
|---------|-------------|
| `intentra install [tool]` | Install hooks for AI tools (cursor, claude, gemini, copilot, windsurf, all) |
| `intentra uninstall [tool]` | Remove hooks from AI tools |
| `intentra hooks status` | Check hook installation status |
| `intentra login` | Authenticate with intentra.sh |
| `intentra logout` | Clear authentication |
| `intentra status` | Show authentication status |
| `intentra scan list` | List captured scans |
| `intentra scan show <id>` | Show scan details |
| `intentra scan today` | List today's scans |
| `intentra config show` | Display configuration |
| `intentra config init` | Generate sample config |
| `intentra config validate` | Validate configuration |

## Supported Tools

| Tool | Status | Hook Format |
|------|--------|-------------|
| Cursor | Supported | camelCase |
| Claude Code | Supported | PascalCase |
| GitHub Copilot | Supported | camelCase |
| Windsurf | Supported | snake_case |
| Gemini CLI | Supported | PascalCase |

## Event Normalization

The CLI normalizes tool-specific hook events into a unified snake_case format. Each tool has its own normalizer in `internal/hooks/`:

```
Native Event → normalizer_<tool>.go → NormalizedType (snake_case)
```

Key normalized event types:
- `before_prompt` / `after_response` - Prompt-response cycle
- `before_tool` / `after_tool` - Generic tool execution
- `before_file_edit` / `after_file_edit` - File operations
- `before_shell` / `after_shell` - Shell commands
- `stop` / `session_end` - Scan boundaries

See `internal/hooks/normalizer.go` for the full list of normalized types.

## Configuration

Configuration file location: `~/.config/intentra/config.yaml`

### Local-Only Mode (Default)

```yaml
server:
  enabled: false
```

### Server Mode (Advanced Observability)

Connect to [intentra.sh](https://intentra.sh) for dashboards, team analytics, and centralized monitoring:

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

## SaaS Features (intentra.sh)

When connected to [intentra.sh](https://intentra.sh), you get access to additional features not available in local-only mode:

### Free Tier
- 100 scans/month synced to cloud
- Basic web dashboard
- Email notifications (daily summary, violations)

### Pro Tier ($20/mo)
- Unlimited scans
- **AI Efficiency Insights** - productivity metrics, benchmarks, forecasting
- **Custom Webhooks** - send notifications to Slack, Discord, or any URL
- **Weekly Digest** - optimization tips delivered to your inbox
- Export evidence for refund claims

### Enterprise
- Team analytics dashboard
- **Audit Logs** - track all UI actions for compliance
- **Audit Log Streaming** - send audit events to your SIEM via webhook
- SSO & RBAC
- SLA & priority support

See [intentra.sh/pricing](https://intentra.sh/#pricing) for full details.

## Documentation

Full documentation: [docs.intentra.sh](https://docs.intentra.sh)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[Apache License 2.0](LICENSE)
