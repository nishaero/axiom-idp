# Security Policy

## Reporting Security Vulnerabilities

If you discover a security vulnerability in Axiom IDP, please do NOT open a public issue. Instead, email security@axiom-idp.dev with the following information:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if available)

We will acknowledge your email within 24 hours and provide a detailed response within 72 hours.

## Security Best Practices

### Environment Variables

Never commit sensitive information. Use `.env` files (add to `.gitignore`):

```env
OAUTH_CLIENT_SECRET=***
API_KEYS=***
DATABASE_PASSWORD=***
```

### Authentication

- Always use HTTPS in production
- Enable OIDC/OAuth2 for user authentication
- Rotate API keys regularly
- Never share authentication tokens

### Rate Limiting

Production deployments should configure rate limiting:

```env
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60s
```

### Database Security

- Use strong passwords for database servers
- Enable database encryption at rest
- Regular backups with encryption
- Use connection pooling with SSL/TLS

### MCP Server Security

When running external MCP servers:

1. Validate server signatures
2. Run in isolated containers
3. Limit server capabilities via configuration
4. Monitor server logs for suspicious activity

## Vulnerability Scanning

We use:

- **Trivy**: Container image scanning
- **Gosec**: Go source code analysis
- **GitHub Actions**: Automated security checks

Run locally:

```bash
trivy image axiom:latest
gosec ./...
```

## Security Headers

Default security headers enabled:

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000
Content-Security-Policy: default-src 'self'
```

## Updates

Always keep Axiom IDP and dependencies updated:

```bash
go get -u ./...
npm audit fix --audit-level=moderate
```

## Access Control

Implement least privilege:

- Use RBAC for user permissions
- Enable audit logging
- Regular access reviews
- Revoke unused credentials

## Compliance

Axiom IDP default security measures:

- ✅ HTTPS/TLS encryption
- ✅ OIDC/OAuth2 authentication
- ✅ RBAC authorization
- ✅ Audit logging
- ✅ Input validation
- ✅ SQL injection prevention
- ✅ CSRF protection
- ✅ Rate limiting

## Deployment Security

### Docker

Use security-hardened base images:

```dockerfile
FROM golang:1.21-alpine AS builder
FROM alpine:latest
RUN addgroup -g 1000 axiom && adduser -D -u 1000 -G axiom axiom
USER axiom
```

### Kubernetes

Apply security policies:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  readOnlyRootFilesystem: true
```

## Incident Response

If a vulnerability is discovered:

1. Assessment & severity classification
2. Patch development & testing
3. Coordinated disclosure
4. Release & notification
5. Post-incident review

## Security Contacts

- Security Team: security@axiom-idp.dev
- Issues: via private vulnerability report
- PGP: See SECURITY.md

## Third-Party Dependencies

We monitor all dependencies for vulnerabilities. Check:

```bash
go list -json -m all | nancy sleuth
npm audit
```
