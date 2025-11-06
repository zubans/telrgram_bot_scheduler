.PHONY: deps migrate-up migrate-down run lint fmt build clean help

DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_PASSWORD ?= postgres
DB_NAME ?= telegram_forwarder
DB_SSLMODE ?= disable

MIGRATE_DSN = postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

GOPATH_BIN = $(shell go env GOPATH)/bin
ifeq ($(GOPATH_BIN),/bin)
    GOPATH_BIN = $(HOME)/go/bin
endif
PATH := $(GOPATH_BIN):$(PATH)
export PATH

deps:
	go mod download
	go mod tidy

migrate-up:
	@if ! command -v migrate > /dev/null 2>&1 && [ ! -f "$(GOPATH_BIN)/migrate" ]; then \
		echo "migrate tool not found. Installing..."; \
		go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
	fi
	@MIGRATE_CMD=$$(command -v migrate 2>/dev/null || echo "$(GOPATH_BIN)/migrate"); \
	if [ ! -f "$$MIGRATE_CMD" ]; then \
		echo "Error: migrate tool not found. Please install it manually: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		exit 1; \
	fi; \
	$$MIGRATE_CMD -path migrations -database "$(MIGRATE_DSN)" up

migrate-down:
	@if ! command -v migrate > /dev/null 2>&1 && [ ! -f "$(GOPATH_BIN)/migrate" ]; then \
		echo "migrate tool not found. Installing..."; \
		go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
	fi
	@MIGRATE_CMD=$$(command -v migrate 2>/dev/null || echo "$(GOPATH_BIN)/migrate"); \
	if [ ! -f "$$MIGRATE_CMD" ]; then \
		echo "Error: migrate tool not found. Please install it manually: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		exit 1; \
	fi; \
	$$MIGRATE_CMD -path migrations -database "$(MIGRATE_DSN)" down

migrate-create:
	@if ! command -v migrate > /dev/null 2>&1 && [ ! -f "$(GOPATH_BIN)/migrate" ]; then \
		echo "migrate tool not found. Installing..."; \
		go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
	fi
	@MIGRATE_CMD=$$(command -v migrate 2>/dev/null || echo "$(GOPATH_BIN)/migrate"); \
	if [ ! -f "$$MIGRATE_CMD" ]; then \
		echo "Error: migrate tool not found. Please install it manually: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		exit 1; \
	fi; \
	read -p "Enter migration name: " name; \
	$$MIGRATE_CMD create -ext sql -dir migrations -seq $$name

run:
	go run cmd/forwarder/main.go

run-once:
	go run cmd/forwarder/main.go -once

build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o forwarder ./cmd/forwarder

lint:
	@if ! command -v golangci-lint > /dev/null; then \
		echo "golangci-lint not found. Installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.55.2; \
	fi
	golangci-lint run ./...

fmt:
	go fmt ./...
	@if ! command -v goimports > /dev/null; then \
		echo "goimports not found. Installing..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	goimports -w .

fmt-check:
	go fmt ./...
	@if ! command -v goimports > /dev/null; then \
		echo "goimports not found. Installing..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@if goimports -l . | grep -q .; then \
		echo "Code is not formatted. Run 'make fmt' to fix."; \
		goimports -l .; \
		exit 1; \
	fi

test:
	go test -v ./...

clean:
	rm -f forwarder
	go clean

help:
	@echo "Available targets:"
	@echo "  deps          - Download and tidy Go dependencies"
	@echo "  migrate-up    - Run database migrations up"
	@echo "  migrate-down  - Rollback database migrations"
	@echo "  migrate-create - Create a new migration file"
	@echo "  run           - Run the application"
	@echo "  run-once      - Run the application once (no scheduler)"
	@echo "  build         - Build the application binary"
	@echo "  lint          - Run linter (golangci-lint)"
	@echo "  fmt           - Format code with gofmt and goimports"
	@echo "  fmt-check     - Check if code is formatted"
	@echo "  test          - Run tests"
	@echo "  clean         - Remove build artifacts"
	@echo "  help          - Show this help message"
	@echo ""
	@echo "Database connection variables (can be overridden):"
	@echo "  DB_HOST       - Database host (default: localhost)"
	@echo "  DB_PORT       - Database port (default: 5432)"
	@echo "  DB_USER       - Database user (default: postgres)"
	@echo "  DB_PASSWORD   - Database password (default: postgres)"
	@echo "  DB_NAME       - Database name (default: telegram_forwarder)"
	@echo "  DB_SSLMODE    - SSL mode (default: disable)"

