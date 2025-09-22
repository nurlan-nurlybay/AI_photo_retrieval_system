# ---- Config ----
APP_NAME       ?= photo-backend
CMD_DIR        ?= ./cmd/server
BIN_DIR        ?= ./bin
BIN_PATH       := $(BIN_DIR)/$(APP_NAME)

# External services (override in .env or CLI)
VECTOR_URL     ?= http://localhost:8002
ML_URL         ?= http://localhost:8001
PORT           ?= 8080

# Database
DB_DSN         ?= postgres://user:pass@localhost:5432/ai_photo?sslmode=disable
MIGRATIONS_DIR ?= ./migrations

# Build metadata
VERSION        := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT         := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE           := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS        := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Load .env if exists
ifneq (,$(wildcard .env))
include .env
export $(shell sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)=.*/\1/p' .env)
endif

# ---- Helpers ----
GO       ?= go
GOTEST   ?= $(GO) test
GOMOD    ?= $(GO) mod
GOLINT   ?= golangci-lint
AIR      ?= air
MIGRATE  ?= migrate

.PHONY: help deps tidy fmt vet lint test cover build run dev clean tools migrate-up migrate-down migrate-new db-url

help:
	@echo "Targets: deps tidy fmt vet lint test cover build run dev migrate-up migrate-down migrate-new clean"

deps:
	$(GOMOD) download

tidy:
	$(GOMOD) tidy

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint:
	@command -v $(GOLINT) >/dev/null 2>&1 && $(GOLINT) run ./... || echo 'golangci-lint not installed; skipping'

test:
	$(GOTEST) ./...

cover:
	$(GOTEST) ./... -coverprofile=coverage.out
	@$(GO) tool cover -func=coverage.out | tail -n 1

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_PATH) $(CMD_DIR)

run:
	PORT=$(PORT) \
	VECTOR_SERVICE_URL=$(VECTOR_URL) \
	ML_SERVICE_URL=$(ML_URL) \
	DB_DSN="$(DB_DSN)" \
	$(GO) run -ldflags "$(LDFLAGS)" $(CMD_DIR)

dev:
	@if command -v $(AIR) >/dev/null 2>&1; then \
		PORT=$(PORT) VECTOR_SERVICE_URL=$(VECTOR_URL) ML_SERVICE_URL=$(ML_URL) DB_DSN="$(DB_DSN)" $(AIR); \
	else \
		echo "air not found; falling back to go run (no hot reload)"; \
		$(MAKE) run; \
	fi

clean:
	@rm -rf $(BIN_DIR) coverage.out

# ---- Migrations (golang-migrate) ----
db-url:
	@echo "$(DB_DSN)"

migrate-up:
	@command -v $(MIGRATE) >/dev/null 2>&1 || { echo "install: https://github.com/golang-migrate/migrate"; exit 1; }
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" up

migrate-down:
	@command -v $(MIGRATE) >/dev/null 2>&1 || { echo "install: https://github.com/golang-migrate/migrate"; exit 1; }
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" down 1

migrate-new:
	@test -n "$(name)" || (echo "usage: make migrate-new name=add_whatever"; exit 2)
	@mkdir -p $(MIGRATIONS_DIR)
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)
