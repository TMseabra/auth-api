# auth-api

Secure authentication API in Go: registration, login, JWT with refresh tokens and roles (`user` and `admin`). First built, then attacked and hardened.

> Status: work in progress (portfolio project).

## Goal

Build a complete identity service and then secure it, documenting every security decision. The story of the project is "I built it, then I secured it".

## Stack

- Go (net/http or chi)
- PostgreSQL (pgx and golang-migrate), running in Docker
- argon2id for password hashing
- JWT (golang-jwt/jwt v5)
- GitHub Actions, Dependabot, Trivy and OWASP ZAP

## Planned features

- Registration and login with argon2id password hashing
- Short-lived access JWT (15 min) and refresh tokens with rotation
- `user` and `admin` roles, with routes protected by middleware
- Login rate limiting and generic error messages

## Planned routes

- GET /healthz (public)
- POST /auth/register (public)
- POST /auth/login (public)
- POST /auth/refresh (refresh token)
- POST /auth/logout (authenticated)
- GET /me (user or admin)
- GET /admin/users (admin only)
- PATCH /admin/users/{id}/role (admin only)

## Plan

### Part 1: Build

1. Go module and /healthz endpoint
2. Environment-based config and docker-compose with PostgreSQL
3. Migrations for the users and refresh_tokens tables
4. Registration with argon2id and input validation
5. Login with access JWT
6. Refresh token rotation and logout
7. Authentication and role middleware
8. Unit and integration tests
9. Login rate limiting

### Part 2: Secure

10. Multi-stage Dockerfile with a distroless image and a non-root user
11. Dependabot (gomod, docker and github-actions)
12. GitHub Actions pipeline: tests, image build and Trivy failing on HIGH and CRITICAL vulnerabilities
13. Initial OWASP ZAP report
14. Fixes, one commit per issue, and a final ZAP report
15. Login flow diagram and a "Security decisions" section

## Security decisions

To be filled in as the project progresses (why argon2id, why refresh token rotation, why distroless, etc.).

## How to run

Requirements: Go and Docker.

Start PostgreSQL (listens on 127.0.0.1:5432 only):

```bash
docker compose up -d
```

Configuration is read from environment variables (see `internal/config`):

| Variable       | Required | Description                                  |
| -------------- | -------- | -------------------------------------------- |
| `PORT`         | no       | HTTP port (default `8080`)                   |
| `DATABASE_URL` | yes      | PostgreSQL connection string                 |
| `JWT_SECRET`   | yes      | Signing secret, at least 32 characters       |

Run the tests:

```bash
go test ./...
```

Current status: steps 1 and 2 of the plan are in place (`/healthz` endpoint, config loader and docker-compose with PostgreSQL). Wiring the config into the server starts in the next steps.

## License

MIT
