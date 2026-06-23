include .env

DATABASE_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

.PHONY: dev test migrate-create migrate-up migrate-down migrate-force migrate-version

dev:
	$$(go env GOPATH)/bin/air

test:
	GOCACHE=$(CURDIR)/.gocache go test ./...

migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=create_table_name"; exit 1; fi
	@command -v migrate >/dev/null 2>&1 || { echo "migrate CLI is not installed. Install it with: brew install golang-migrate"; exit 1; }
	migrate create -ext sql -dir migrations -seq $(name)

migrate-up:
	@command -v migrate >/dev/null 2>&1 || { echo "migrate CLI is not installed. Install it with: brew install golang-migrate"; exit 1; }
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	@command -v migrate >/dev/null 2>&1 || { echo "migrate CLI is not installed. Install it with: brew install golang-migrate"; exit 1; }
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-force:
	@if [ -z "$(version)" ]; then echo "Usage: make migrate-force version=1"; exit 1; fi
	@command -v migrate >/dev/null 2>&1 || { echo "migrate CLI is not installed. Install it with: brew install golang-migrate"; exit 1; }
	migrate -path migrations -database "$(DATABASE_URL)" force $(version)

migrate-version:
	@command -v migrate >/dev/null 2>&1 || { echo "migrate CLI is not installed. Install it with: brew install golang-migrate"; exit 1; }
	migrate -path migrations -database "$(DATABASE_URL)" version
