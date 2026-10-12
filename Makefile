SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

-include .env
LOG_FORMAT ?= text
export DATABASE_URL ADDR SESSION_TTL COOKIE_SECURE CORS_ORIGIN TRUSTED_PROXIES RATE_LIMIT_PER_MINUTE LOG_FORMAT
POSTGRES_USER     ?= portfolio
POSTGRES_PASSWORD ?= portfolio
POSTGRES_DB       ?= portfolio
POSTGRES_PORT     ?= 5432
TEST_DB           := $(POSTGRES_DB)_test
E2E_DB            := $(POSTGRES_DB)_e2e
XLSX              ?= $(HOME)/Desktop/Portfolio-Tracker.xlsx
PG_DIR            := backend/internal/adapter/postgres
GEN_DIR           := $(PG_DIR)/internal/gen
SQLC_VERSION      := 1.31.1

BIN         := backend/portfolio-tracker$(if $(filter Windows_NT,$(OS)),.exe,)
GOVULNCHECK := golang.org/x/vuln/cmd/govulncheck@v1.9.0
PSQL        := docker compose exec -T db psql -v ON_ERROR_STOP=1 -U $(POSTGRES_USER) -d $(POSTGRES_DB)

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: install
install: ## Install Go modules and npm packages
	cd backend && go mod download
	cd web && npm ci

.PHONY: db-up db-down db-migrate db-reset db-psql
db-up: ## Start PostgreSQL and wait until it is healthy
	docker compose up -d --wait db

db-down: ## Stop PostgreSQL (data is kept)
	docker compose stop db

db-migrate: db-up ## Apply pending migrations without starting the server
	cd backend && go run ./cmd/server migrate

db-reset: db-up ## DELETE ALL DATA, then re-run migrations (asks first; FORCE=1 skips)
	@if [ "$(FORCE)" != "1" ]; then \
		read -r -p "Drop every table in '$(POSTGRES_DB)'? This cannot be undone. Type 'yes': " ans; \
		[ "$$ans" = "yes" ] || { echo "aborted"; exit 1; }; \
	fi
	$(PSQL) -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	$(MAKE) --no-print-directory db-migrate
	@echo "Database reset. Start the server to seed admin/admin."

db-psql: db-up ## Open a psql shell
	docker compose exec db psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

.PHONY: seed-sql
seed-sql: ## Generate private/seed-data.sql from $(XLSX) (git-ignored, run once on a fresh DB)
	cd backend && go run ./cmd/xlsx2sql -in "$(XLSX)" -out ../private/seed-data.sql

.PHONY: db-seed
db-seed: seed-sql db-migrate ## Load $(XLSX) into the local DB (tables must be empty: run db-reset first)
	$(PSQL) -q < private/seed-data.sql
	@echo "Seeded $(POSTGRES_DB) from $(XLSX)"

.PHONY: e2e-seed
e2e-seed: db-up ## Recreate $(POSTGRES_DB)_e2e with made-up data from backend/testdata/e2e-seed.sql
	docker compose exec -T db sh -c 'dropdb -U $(POSTGRES_USER) --if-exists $(E2E_DB) && createdb -U $(POSTGRES_USER) $(E2E_DB)'
	cd backend && DATABASE_URL="postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(E2E_DB)?sslmode=disable" go run ./cmd/server migrate
	docker compose exec -T db psql -v ON_ERROR_STOP=1 -U $(POSTGRES_USER) -d $(E2E_DB) -q < backend/testdata/e2e-seed.sql
	@echo "Seeded $(E2E_DB): log in as e2e / E2e-Tiger-Mango-42, or admin / admin for the forced password change"

SWAGGER_UI_VERSION := 5.33.0
SWAGGER_UI_DIR     := backend/internal/adapter/http/openapi/swagger-ui

.PHONY: swagger-ui
swagger-ui: ## Vendor swagger-ui-dist $(SWAGGER_UI_VERSION) (bundle + css) for /api/docs
	tmp=$$(mktemp -d); (cd "$$tmp" && npm pack -s swagger-ui-dist@$(SWAGGER_UI_VERSION) >/dev/null && tar xzf swagger-ui-dist-$(SWAGGER_UI_VERSION).tgz); 		mkdir -p $(SWAGGER_UI_DIR) && cp "$$tmp"/package/{swagger-ui-bundle.js,swagger-ui.css,LICENSE,NOTICE} $(SWAGGER_UI_DIR)/ && rm -rf "$$tmp"
	echo $(SWAGGER_UI_VERSION) > $(SWAGGER_UI_DIR)/VERSION

.PHONY: migration
migration: ## Create an empty migration: make migration name=add_fund_tags
	@[[ "$(name)" =~ ^[a-z0-9_]+$$ ]] || { echo "name must be snake_case, e.g. make migration name=add_fund_tags"; exit 1; }
	@f=$(PG_DIR)/migrations/$$(date -u +%Y%m%d%H%M%S)_$(name).sql; \
		( set -C; printf -- '-- %s\n' "$(name)" > "$$f" ) 2>/dev/null || { echo "$$f exists; wait a second and retry"; exit 1; }; \
		echo "created $$f"

.PHONY: sqlc sqlc-check
# sqlc needs cgo, which is unreliable on Windows, so it runs from its pinned
# image everywhere. pwd -W gives Docker Desktop a C:/ path under Git Bash, and
# MSYS_NO_PATHCONV stops Git Bash rewriting /src.
sqlc: ## Regenerate Go code from migrations/ and queries/ with sqlc
	MSYS_NO_PATHCONV=1 docker run --rm -v "$$(cd $(PG_DIR) && (pwd -W 2>/dev/null || pwd)):/src" -w /src sqlc/sqlc:$(SQLC_VERSION) generate

# Generating into an empty dir also catches orphan files left by a deleted query.
sqlc-check: ## Fail when the committed sqlc output differs from a fresh generate
	rm -rf $(GEN_DIR)
	$(MAKE) --no-print-directory sqlc
	@if [ -n "$$(git status --porcelain -- $(GEN_DIR))" ]; then \
		git status --short -- $(GEN_DIR); git diff -- $(GEN_DIR); \
		echo "sqlc output is stale: run 'make sqlc' and commit $(GEN_DIR)"; exit 1; \
	fi

.PHONY: app-up app-down app-logs
app-up: ## Build and start db, api and web in the background on http://localhost:$${APP_PORT:-8000}
	docker compose --profile app up -d --build --wait
	@echo "App: http://localhost:$${APP_PORT:-8000}"

app-down: ## Stop api and web (the db keeps running; db-down stops it)
	docker compose --profile app stop api web

app-logs: ## Follow api and web logs
	docker compose --profile app logs -f api web

.PHONY: dev-api dev-web
dev-api: db-up ## Run the backend on :8080
	cd backend && go run ./cmd/server

dev-web: ## Run the Vite dev server on :5173 (proxies /api to :8080)
	cd web && npm run dev

.PHONY: test test-integration vet typecheck lint vulncheck check
# -p 1: the postgres and app packages both reset the database behind TEST_DATABASE_URL.
test: ## Run Go tests (CI adds GOTESTFLAGS=-race)
	cd backend && go test -p 1 $(GOTESTFLAGS) ./...

test-integration: db-up ## Run the Postgres and API tests in a fresh $(POSTGRES_DB)_test database
	docker compose exec -T db sh -c 'dropdb -U $(POSTGRES_USER) --if-exists $(TEST_DB) && createdb -U $(POSTGRES_USER) $(TEST_DB)'
	cd backend && TEST_DATABASE_URL="postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(TEST_DB)?sslmode=disable" 		go test -p 1 $(GOTESTFLAGS) -count=1 ./internal/adapter/postgres/ ./internal/app/

vet: ## go vet + gofmt check
	cd backend && go vet ./...
	@out=$$(cd backend && gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

typecheck: ## TypeScript type check
	cd web && npm run typecheck

lint: sqlc-check vet typecheck ## All static checks

vulncheck: ## Scan Go (govulncheck) and npm (npm audit) dependencies for known vulnerabilities
	cd backend && go run $(GOVULNCHECK) ./...
	cd web && npm audit --audit-level=moderate
	cd e2e && npm audit --audit-level=moderate

check: lint test vulncheck ## Everything CI runs

.PHONY: build-web build run clean
build-web: ## Build the SPA into web/dist
	cd web && npm run build

build: ## Build the API binary (serves only /api unless WEB_DIR is set)
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(notdir $(BIN)) ./cmd/server
	@echo "built $(BIN)"

run: build build-web db-up ## Build both and run the binary, serving web/dist too
	cd backend && WEB_DIR=../web/dist ./$(notdir $(BIN))

clean: ## Remove build output
	rm -rf $(BIN) web/dist

E2E         := docker compose -f e2e/docker-compose.yml
# Windows drives the installed Edge; elsewhere Playwright downloads Chromium.
PLAYWRIGHT  := node node_modules/@playwright/test/cli.js
E2E_CHANNEL ?= $(if $(filter Windows_NT,$(OS)),msedge,)
export E2E_CHANNEL

.PHONY: e2e e2e-up e2e-ui e2e-down docs-screenshots
e2e-up: ## Build and start the throwaway e2e stack (web :8100, in-memory db :55432)
	$(E2E) up -d --build --wait
	cd e2e && npm ci
	$(if $(E2E_CHANNEL),,cd e2e && $(PLAYWRIGHT) install chromium)

e2e: e2e-up ## Run the Playwright suite on made-up data, then stop the stack
	status=0; (cd e2e && $(PLAYWRIGHT) test) || status=$$?; $(E2E) down; exit $$status

e2e-ui: e2e-up ## Open Playwright UI mode against the e2e stack (stop it with e2e-down)
	cd e2e && $(PLAYWRIGHT) test --ui

docs-screenshots: e2e-up ## Regenerate docs/images from the made-up e2e seed, then stop the stack
	status=0; (cd e2e && UPDATE_SCREENSHOTS=1 $(PLAYWRIGHT) test tests/docs-screenshots.spec.ts) || status=$$?; $(E2E) down; exit $$status

e2e-down: ## Stop the e2e stack and drop its data
	$(E2E) down

DEPLOY := docker compose -f deploy/docker-compose.yml

.PHONY: docker-build deploy-up deploy-down
docker-build: ## Build the api and web Docker images
	docker build -t portfolio-tracker-api backend
	docker build -t portfolio-tracker-web web

deploy-up: ## Build and start db, api and web from deploy/ (needs deploy/.env)
	$(DEPLOY) up -d --build --wait

deploy-down: ## Stop the deploy stack (data is kept)
	$(DEPLOY) down
