# Contributing

1. Start PostgreSQL: `make db`
2. Create the test database once:
   `docker compose exec db psql -U auth -d auth -c 'CREATE DATABASE auth_test;'`
3. Run the checks before opening a pull request: `make fmt vet test`

Keep pull requests small and focused. Security-sensitive changes (anything under
`internal/auth` or `internal/httpapi`) need a test that fails without the change.
