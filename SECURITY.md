# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to: **security@intentra.io**

Include the following information:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### What to Expect

- **Acknowledgment**: We will acknowledge receipt within 48 hours
- **Assessment**: We will assess the vulnerability and determine its severity
- **Updates**: We will keep you informed of our progress
- **Resolution**: We aim to resolve critical vulnerabilities within 7 days
- **Credit**: With your permission, we will credit you in the security advisory

### Scope

The following are in scope:

- Intentra CLI application code
- Hook installation and execution
- Data handling and storage
- Network communication (when server sync is enabled)
- Authentication mechanisms

### Out of Scope

- Vulnerabilities in third-party dependencies (report to the respective projects)
- Issues in AI tools themselves (Cursor, Claude Code, etc.)
- Social engineering attacks

## Security Best Practices

When using Intentra CLI:

1. **Protect your configuration file** - It may contain API credentials
   ```bash
   chmod 600 ~/.config/intentra/config.yaml
   ```

2. **Use HMAC authentication** - When connecting to a server, prefer HMAC over simple API keys

3. **Review hook permissions** - Understand what data hooks collect before installing

4. **Keep updated** - Always use the latest version for security patches

## Disclosure Policy

We follow responsible disclosure:

1. Security issues are fixed in private
2. A fix is prepared and tested
3. A new version is released
4. The vulnerability is disclosed publicly after users have had time to update
