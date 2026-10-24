# Changelog

## 1.0.0

- Registration and login with argon2id password hashing
- Access JWT (15 min) and rotating refresh tokens with reuse detection
- `user` and `admin` roles with protected routes
- Per-IP and per-account rate limiting
- Distroless non-root image, Dependabot, CI with Trivy
- Security fixes found by Trivy and OWASP ZAP (see `docs/security/`)
