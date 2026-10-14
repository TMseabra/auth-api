.PHONY: db run test fmt vet

TEST_DATABASE_URL ?= postgres://auth:auth_dev_password@localhost:5432/auth_test?sslmode=disable

db:
	docker compose up -d --wait

run:
	go run ./cmd/api

test:
	TEST_DATABASE_URL=$(TEST_DATABASE_URL) go test ./... -count=1 -p 1

fmt:
	gofmt -w cmd internal migrations

vet:
	go vet ./...
