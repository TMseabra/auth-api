# auth-api

[![CI](https://github.com/TMseabra/auth-api/actions/workflows/ci.yml/badge.svg)](https://github.com/TMseabra/auth-api/actions/workflows/ci.yml)

Secure authentication API in Go: registration, login, JWT with refresh tokens and roles (`user` and `admin`). First built, then attacked and hardened.

> Status: all 15 steps of the plan are done. Portfolio project, not production-reviewed.

## Goal

Build a complete identity service and then secure it, documenting every security decision. The story of the project is "I built it, then I secured it".

## Stack

- Go (net/http)
- PostgreSQL (pgx and golang-migrate), running in Docker
- argon2id for password hashing
- JWT (golang-jwt/jwt v5)
- GitHub Actions, Dependabot, Trivy and OWASP ZAP

## Features

- Registration and login with argon2id password hashing
- Short-lived access JWT (15 min) and refresh tokens (7 days) with rotation and reuse detection
- `user` and `admin` roles, with routes protected by middleware
- Login rate limiting (per IP and per account) and generic error messages

## Routes

| Route                            | Access                                    |
| -------------------------------- | ----------------------------------------- |
| `GET /healthz`                   | public                                    |
| `POST /auth/register`            | public                                    |
| `POST /auth/login`               | public                                    |
| `POST /auth/refresh`             | refresh token in the body                 |
| `POST /auth/logout`              | authenticated, revokes all refresh tokens |
| `GET /me`                        | user or admin                             |
| `GET /admin/users`               | admin only                                |
| `PATCH /admin/users/{id}/role`   | admin only                                |

The full contract is in [docs/openapi.yaml](docs/openapi.yaml).

## Login flow

```mermaid
sequenceDiagram
    participant C as Client
    participant A as auth-api
    participant D as PostgreSQL

    C->>A: POST /auth/login {email, password}
    A->>A: per-IP and per-account rate limit
    A->>D: find user by email
    A->>A: verify argon2id hash (dummy hash if user unknown)
    A->>D: store SHA-256 of a new refresh token
    A-->>C: access JWT (15 min) + refresh token (7 days)

    C->>A: GET /me (Authorization: Bearer access JWT)
    A->>A: check signature, HS256, issuer, expiry
    A-->>C: user

    C->>A: POST /auth/refresh {refresh_token}
    A->>D: revoke old token, store new one (one transaction)
    A-->>C: new access JWT + new refresh token

    Note over C,A: Presenting an already-used refresh token revokes<br/>every refresh token of that user (reuse detection)
```

## Project layout

```
cmd/api/           entry point (config, migrations, HTTP server, graceful shutdown)
internal/auth/     argon2id passwords, access JWTs, refresh token generation
internal/config/   environment configuration
internal/db/       connection pool and embedded migrations runner
internal/httpapi/  handlers, middleware, rate limiter, security headers
internal/store/    SQL queries for users and refresh tokens
migrations/        versioned SQL migrations
docs/              OpenAPI spec and security reports
```

## Plan

### Part 1: Build

1. Go module and /healthz endpoint
2. Environment-based config and docker-compose with PostgreSQL
3. Migrations for the users and refresh_tokens tables
4. Registration with argon2id and input validation
5. Login with access JWT
6. Refresh token rotation (logout arrived with the middleware in step 7)
7. Authentication and role middleware
8. Unit and integration tests
9. Login rate limiting

### Part 2: Secure

10. Multi-stage Dockerfile with a distroless image and a non-root user
11. Dependabot (gomod, docker and github-actions)
12. GitHub Actions pipeline: tests, image build and Trivy failing on HIGH and CRITICAL vulnerabilities
13. Initial OWASP ZAP report: [docs/security/zap-initial.md](docs/security/zap-initial.md)
14. Fixes, one commit per issue, and a final ZAP report: [docs/security/zap-final.md](docs/security/zap-final.md)
15. Login flow diagram and a "Security decisions" section

## Security decisions

**Passwords: argon2id** (64 MiB, 3 passes, 2 lanes, random 16-byte salt, PHC string format). It is memory-hard, which makes GPU cracking expensive, and it is the OWASP first choice. Passwords must have 12 to 128 characters (the upper bound stops someone from making the server hash megabytes of input).

**Generic login errors and equal timing.** An unknown email and a wrong password return the same `401 invalid credentials`. For unknown emails the server still computes an argon2id hash, so response time does not reveal which emails exist. Registration does return `409` for a taken email; hiding it would need an email-verification flow, which is out of scope.

**Short access token, opaque refresh token.** The access JWT lives 15 minutes and is verified with a fixed algorithm list (HS256 only, so `alg: none` and algorithm-confusion tokens are rejected), the issuer and a required expiry. The refresh token is 32 random bytes; only its SHA-256 is stored, so a database leak does not leak usable tokens.

**Refresh token rotation with reuse detection.** Every refresh revokes the presented token and issues a new one inside one transaction (`SELECT ... FOR UPDATE`, so two concurrent refreshes cannot both win). If a revoked token shows up again it is probably stolen, so all the user's refresh tokens are revoked and the legitimate user has to log in again.

**Rate limiting in two layers.** A per-IP bucket on register, login and refresh (they are CPU and memory heavy because of argon2id), and a per-account bucket on login against credential stuffing spread over many IPs. Trade-off: an attacker can throttle a victim's logins (not their existing sessions). The client IP is the TCP peer address; `X-Forwarded-For` is ignored because clients can forge it.

**Input handling.** JSON bodies are limited to 1 MiB, unknown fields are rejected (so `{"role":"admin"}` at registration is a `400`, not a privilege escalation), and ids are validated as UUIDs before they reach SQL. All queries are parameterized.

**Roles.** Role checks happen in middleware. An admin cannot change their own role (avoids locking the last admin out). The role is read from the access token, so a demotion takes effect within 15 minutes. The first admin is created in the database (see below).

**Distroless, non-root image.** The final image has no shell and no package manager, runs as `nonroot`, and the compose service also uses a read-only filesystem, drops all capabilities and sets `no-new-privileges`.

**Supply chain.** Dependabot watches Go modules, the Dockerfile and the workflow actions. CI runs the tests against a real PostgreSQL and fails when Trivy finds a fixable HIGH or CRITICAL vulnerability.

**Security headers.** Every response sends `X-Content-Type-Options: nosniff` and `Cross-Origin-Resource-Policy: same-origin`.

### What the scans found

| Tool  | Finding                                                                 | Fix                                  |
| ----- | ----------------------------------------------------------------------- | ------------------------------------ |
| Trivy | CVE-2026-56854 (HIGH) in `golang.org/x/crypto` v0.53.0, `ssh` package   | Bumped to v0.58.0, rescan is clean   |
| ZAP   | X-Content-Type-Options header missing (low)                             | Header added on every response       |
| ZAP   | Cross-Origin-Resource-Policy header missing (low)                       | Header added on every response       |

ZAP went from 2 warnings to 0 (119 passing checks). Limits of that scan: it is an unauthenticated API scan driven by the OpenAPI file, and the rate limiter throttles it, so protected routes were mostly exercised as `401`. It is evidence, not proof.

### Known limitations

- No TLS in the service: run it behind a reverse proxy that terminates HTTPS.
- Rate-limit state is in memory, so it is per process.
- No email verification, password reset or account lockout notifications.
- The JWT secret is a shared HS256 secret; rotating it logs everyone out.
- Expired and revoked refresh tokens stay in the database; there is no cleanup job yet.

## How to run

Requirements: Go and Docker.

```bash
cp .env.example .env              # then set JWT_SECRET (at least 32 characters)
docker compose up -d              # PostgreSQL on 127.0.0.1:5432 only
```

Run the API in Docker:

```bash
JWT_SECRET=<your secret> docker compose --profile app up --build
```

or locally (`DATABASE_URL` and `JWT_SECRET` come from the environment; migrations run on startup):

```bash
export DATABASE_URL='postgres://auth:auth_dev_password@localhost:5432/auth?sslmode=disable'
export JWT_SECRET=<your secret>
go run ./cmd/api
```

| Variable       | Required | Description                            |
| -------------- | -------- | -------------------------------------- |
| `PORT`         | no       | HTTP port (default `8080`)             |
| `DATABASE_URL` | yes      | PostgreSQL connection string           |
| `JWT_SECRET`   | yes      | Signing secret, at least 32 characters |

Try it:

```bash
curl -X POST localhost:8080/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"me@example.com","password":"a-long-enough-password"}'
curl -X POST localhost:8080/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"me@example.com","password":"a-long-enough-password"}'
```

Make the first admin (there is deliberately no API for this):

```bash
docker compose exec db psql -U auth -d auth \
  -c "UPDATE users SET role = 'admin' WHERE email = 'me@example.com';"
```

Run the tests. The integration tests need a database and are skipped without `TEST_DATABASE_URL`:

```bash
docker compose exec db psql -U auth -d auth -c 'CREATE DATABASE auth_test;'
export TEST_DATABASE_URL='postgres://auth:auth_dev_password@localhost:5432/auth_test?sslmode=disable'
go test ./... -p 1
```

## Troubleshooting

- `JWT_SECRET is required and must be at least 32 characters`: set a longer secret, for example `openssl rand -hex 32`.
- `connection refused` on port 5432: PostgreSQL is not running. Start Docker and run `docker compose up -d --wait`.
- Integration tests are reported as skipped: `TEST_DATABASE_URL` is not set (see "How to run").
- `429 too many requests` while testing by hand: the login limiter is working. Wait a minute or restart the API, since the limiter state is in memory.

## License

MIT
