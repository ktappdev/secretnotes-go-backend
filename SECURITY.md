# Security Policy

## Supported Versions

We actively support the latest version of Secret Notes Go Backend. Security updates will be provided for:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest| :x:                |

## Security Features

### Encryption Implementation

- **Algorithm**: AES-256-GCM (Galois/Counter Mode)
- **Key Derivation**: PBKDF2 with SHA-256 (100,000 iterations)
- **Salt**: 16-byte random salt per encryption operation
- **Nonce**: 12-byte random nonce per encryption operation
- **Authentication**: Built-in authentication tag with GCM mode

### Data Protection

- **Zero-Knowledge Architecture**: Server cannot decrypt user data without passphrase
- **No Stored Secrets**: Passphrases are never stored, only SHA-256 hashes for lookup
- **Encrypted at Rest**: All user content encrypted before database storage
- **Stateless Design**: No sessions or authentication tokens stored

### Security Best Practices

- **Minimum Passphrase Length**: 32 characters enforced
- **Secure Random Generation**: Uses crypto/rand for all random values
- **Memory Safety**: Go's memory management prevents common vulnerabilities
- **Input Validation**: All inputs validated and sanitized

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please follow these steps:

### 🚨 For Critical Security Issues

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please report security issues by:

1. **Email**: Send details to 'kentaylorappdev@gmail.com'
2. **Subject Line**: Include "SECURITY" in the subject line
3. **Include**:
   - Description of the vulnerability
   - Steps to reproduce the issue
   - Potential impact assessment
   - Suggested fix (if available)

### Response Timeline

- **Acknowledgment**: Within 24 hours
- **Initial Assessment**: Within 72 hours
- **Status Updates**: Every 7 days until resolved
- **Resolution**: Target 30 days for critical issues

### Disclosure Policy

- We follow **responsible disclosure** practices
- Security fixes will be released as soon as possible
- Public disclosure will occur **after** fixes are available
- Credit will be given to researchers who report issues responsibly

## Security Considerations for Deployment

### Production Deployment

1. **HTTPS Only**: Never deploy without TLS/SSL encryption
2. **Reverse Proxy**: Use nginx/Apache with proper security headers
3. **Firewall**: Restrict access to necessary ports only
4. **Updates**: Keep Go and dependencies updated
5. **Monitoring**: Implement logging and intrusion detection

### Infrastructure Security

```bash
# Example nginx configuration
server {
    listen 443 ssl http2;
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    
    location / {
        proxy_pass http://localhost:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Environment Security

- **Environment Variables**: Use for sensitive configuration
- **File Permissions**: Restrict access to data directory (700)
- **User Privileges**: Run with minimal required privileges
- **Network Isolation**: Use containers or VMs when possible

### Backup Security

- **Encrypt Backups**: Use additional encryption for backup files
- **Secure Storage**: Store backups in secure, separate locations
- **Access Control**: Limit backup access to authorized personnel
- **Regular Testing**: Verify backup integrity and restore procedures

## Known Security Considerations

### Passphrase Security

- **User Responsibility**: Users must choose strong passphrases
- **No Recovery**: Lost passphrases mean lost data (by design)
- **Brute Force**: Server-side rate limiting recommended
- **Transmission**: Always use HTTPS to protect passphrases in transit

### Cryptographic Dependencies

- **Go Standard Library**: Uses well-vetted crypto packages
- **Regular Updates**: Monitor for security updates in dependencies
- **Algorithm Agility**: Code designed to support algorithm updates

### Operational Security

- **Log Security**: Ensure logs don't contain sensitive data
- **Memory Dumps**: Consider memory dump protection in production
- **Side Channels**: Be aware of timing attack possibilities
- **Physical Security**: Secure physical access to servers

## Security Audit

This project welcomes security audits from the community. If you're conducting a security review:

1. **Scope**: Focus on cryptographic implementation and data handling
2. **Tools**: Static analysis tools are welcome (gosec, etc.)
3. **Testing**: Penetration testing on your own instances only
4. **Reporting**: Follow the vulnerability reporting process above

## Compliance

This implementation aims to follow:

- **OWASP Top 10**: Protection against common web vulnerabilities
- **Cryptographic Standards**: NIST-approved algorithms and parameters
- **Data Protection**: Privacy-by-design principles

## Security Resources

- [Go Security Policy](https://golang.org/security)
- [OWASP Secure Coding Practices](https://owasp.org/www-project-secure-coding-practices-quick-reference-guide/)
- [NIST Cryptographic Standards](https://csrc.nist.gov/projects/cryptographic-standards-and-guidelines)

---

**Remember**: Security is a shared responsibility between the software, deployment, and usage practices.
