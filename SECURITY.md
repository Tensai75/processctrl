# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability in processctrl, please report it responsibly.

### How to Report

1. **Do not** create a public GitHub issue for security vulnerabilities
2. Send an email to [tensai75@protonmail.com] with:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### What to Expect

- **Acknowledgment**: We'll acknowledge receipt of your report within 48 hours
- **Investigation**: We'll investigate and validate the vulnerability
- **Timeline**: We aim to provide updates every 7 days during investigation
- **Resolution**: We'll work to fix confirmed vulnerabilities promptly
- **Credit**: We'll credit you in the security advisory (unless you prefer to remain anonymous)

### Security Considerations

This package executes external processes and handles:

- Process creation and control
- Input/output streams
- System signals and APIs

Please report any issues related to:

- Command injection vulnerabilities
- Privilege escalation
- Resource exhaustion attacks
- Information disclosure

## Security Best Practices

When using processctrl:

1. **Validate inputs**: Always validate and sanitize command arguments
2. **Limit permissions**: Run processes with minimal required privileges
3. **Handle errors**: Properly handle and log security-related errors
4. **Resource limits**: Set appropriate timeouts and resource limits
5. **Monitor usage**: Log process execution for security auditing

Thank you for helping keep processctrl secure!
