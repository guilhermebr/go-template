OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)
PKG ?= ./cmd
TERM=xterm-256color
CLICOLOR_FORCE=true
RICHGO_FORCE_COLOR=1
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_BUILD_TIME=$(shell date '+%Y-%m-%d__%I:%M:%S%p')
GO_BIN_PATH=$(shell go env GOPATH)/bin

.PHONY: help
help: ## Show available make targets
	@echo "Available targets:"
	@echo ""
	@echo "Build & Generate:"
	@echo "  build              Build all binaries (service, admin, web)"
	@echo "  generate           Generate all code (templ + sqlc + go generate)"
	@echo "  templ             Generate templ templates"
	@echo "  sqlc-generate     Generate sqlc code"
	@echo ""
	@echo "OpenAPI & SDKs:"
	@echo "  docs                      Show documentation URLs"
	@echo "  openapi-generate          Generate OpenAPI spec from Go code"
	@echo "  sdk-go                    Generate Go client SDK (manual spec)"
	@echo "  sdk-typescript            Generate TypeScript client SDK (manual spec)"
	@echo "  sdk-python                Generate Python client SDK (manual spec)"
	@echo "  sdk-java                  Generate Java client SDK (manual spec)"
	@echo "  sdks                      Generate all client SDKs (manual spec)"
	@echo "  sdks-generated            Generate all client SDKs (from code annotations)"
	@echo ""
	@echo "Development:"
	@echo "  setup             Install all required tools"
	@echo "  test              Run tests (short)"
	@echo "  test-full         Run all tests with coverage"
	@echo "  coverage          Generate HTML coverage report"
	@echo "  lint              Run linters"
	@echo "  gosec             Run security analysis"
	@echo ""
	@echo "Database:"
	@echo "  migration/create  Create new migration"
	@echo "  migration/up      Apply migrations"
	@echo "  migration/down    Rollback migrations"

define goBuild
	@echo "==> Go Building $2"
	@env GOOS=${OS} GOARCH=${ARCH} go build -v -o  build/$1 \
	-ldflags "-X main.BuildCommit=$(GIT_COMMIT) -X main.BuildTime=$(GIT_BUILD_TIME)" \
	${PKG}/$2
endef

# Build all binaries (automatically generates templates and code first)
.PHONY: build
build: generate
	$(call goBuild,service,"service")
	$(call goBuild,admin,"admin")
	$(call goBuild,web,"web")

# ###########
# Setup
# ###########

.PHONY: install-moq
install-moq:
	@echo "==> Installing moq"
	@go install github.com/matryer/moq@latest

.PHONY: install-migration
install-migration:
	@echo "==> Installing migration"
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

.PHONY: install-linters
install-linters:
	@echo "==> Installing linters"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: install-test-fmt
install-test-fmt:
	@echo "==> Installing test formatter"
	@go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@latest

.PHONY: install-gosec 
install-gosec:
	@echo "==> Installing gosec"
	@go install github.com/securego/gosec/v2/cmd/gosec@latest

.PHONY: install-sqlc 
install-sqlc:
	@echo "==> Installing sqlc"
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

.PHONY: install-templ
install-templ:
	@echo "==> Installing templ"
	@go install github.com/a-h/templ/cmd/templ@latest

.PHONY: install-oapi-codegen
install-oapi-codegen:
	@echo "==> Installing oapi-codegen"
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

.PHONY: install-swag-v2
install-swag-v2:
	@echo "==> Installing swag"
	@go install github.com/swaggo/swag/cmd/swag@latest

.PHONY: install-openapi-generator
install-openapi-generator:
	@echo "==> Installing OpenAPI Generator"
	@npm install -g @openapitools/openapi-generator-cli


.PHONY: setup
setup: install-migration install-moq install-linters install-test-fmt install-gosec install-sqlc install-templ install-oapi-codegen install-swag-v2 install-openapi-generator
	@go mod tidy


# ###########
# Generate
# ###########

# Generate templ templates from .templ files
.PHONY: templ
templ:
	@echo "==> Generating templ templates"
	@templ generate


# Generate sqlc code
.PHONY: sqlc-generate
sqlc-generate:
	@echo "==> Generating sqlc code"
	@rm -f gateways/repository/pg/gen/*.go
	@sqlc generate

# Generate OpenAPI 3.x spec from Go annotations
.PHONY: openapi-generate
openapi-generate:
	@echo "==> Generating OpenAPI 3.x spec from Go code"
	@${GO_BIN_PATH}/swag init -g cmd/service/main.go -o docs/ --parseDependency --parseInternal
	@mv docs/swagger.yaml docs/openapi-generated.yaml
	@mv docs/swagger.json docs/openapi-generated.json
	@echo "==> Specs generated at docs/openapi-generated.{yaml,json}"

# Generate OpenAPI documentation
.PHONY: docs
docs:
	@echo "==> OpenAPI documentation is ready at:"
	@echo "    - Redoc:      docs/redoc.html"
	@echo "    - Swagger UI: docs/swagger-ui.html"
	@echo "    - Manual:     docs/openapi.yaml"
	@echo "    - Generated:  docs/openapi-generated.yaml"

# Generate Go client SDK using oapi-codegen
.PHONY: sdk-go
sdk-go:
	@echo "==> Generating Go SDK"
	@${GO_BIN_PATH}/oapi-codegen -config configs/oapi-codegen.yaml docs/openapi.yaml

# Generate TypeScript SDK using OpenAPI Generator
.PHONY: sdk-typescript
sdk-typescript:
	@echo "==> Generating TypeScript SDK"
	@npx openapi-generator-cli generate \
		-i docs/openapi.yaml \
		-g typescript-fetch \
		-o sdks/typescript \
		-c configs/openapi-generator-typescript.json

# Generate Python SDK using OpenAPI Generator
.PHONY: sdk-python
sdk-python:
	@echo "==> Generating Python SDK"
	@npx openapi-generator-cli generate \
		-i docs/openapi.yaml \
		-g python \
		-o sdks/python \
		-c configs/openapi-generator-python.json

# Generate Java SDK using OpenAPI Generator
.PHONY: sdk-java
sdk-java:
	@echo "==> Generating Java SDK"
	@npx openapi-generator-cli generate \
		-i docs/openapi.yaml \
		-g java \
		-o sdks/java \
		-c configs/openapi-generator-java.json

# Generate SDKs from generated spec
.PHONY: sdks-generated
sdks-generated: openapi-generate sdk-go-generated sdk-typescript-generated sdk-python-generated sdk-java-generated
	@echo "==> All SDKs generated successfully from code annotations"

# Generate Go SDK from generated spec
.PHONY: sdk-go-generated
sdk-go-generated:
	@echo "==> Generating Go SDK from generated spec"
	@${GO_BIN_PATH}/oapi-codegen -config configs/oapi-codegen.yaml docs/openapi-generated.yaml

# Generate TypeScript SDK from generated spec
.PHONY: sdk-typescript-generated
sdk-typescript-generated:
	@echo "==> Generating TypeScript SDK from generated spec"
	@npx openapi-generator-cli generate \
		-i docs/openapi-generated.yaml \
		-g typescript-fetch \
		-o sdks/typescript \
		-c configs/openapi-generator-typescript.json

# Generate Python SDK from generated spec
.PHONY: sdk-python-generated
sdk-python-generated:
	@echo "==> Generating Python SDK from generated spec"
	@npx openapi-generator-cli generate \
		-i docs/openapi-generated.yaml \
		-g python \
		-o sdks/python \
		-c configs/openapi-generator-python.json

# Generate Java SDK from generated spec
.PHONY: sdk-java-generated
sdk-java-generated:
	@echo "==> Generating Java SDK from generated spec"
	@npx openapi-generator-cli generate \
		-i docs/openapi-generated.yaml \
		-g java \
		-o sdks/java \
		-c configs/openapi-generator-java.json

# Generate all SDKs from manual spec
.PHONY: sdks
sdks: sdk-go sdk-typescript sdk-python sdk-java
	@echo "==> All SDKs generated successfully from manual spec"

# Generate all code (templ templates + sqlc + go generate)
.PHONY: generate
generate: templ sqlc-generate
	@echo "==> Running go generate"
	@go generate ./...


# ###########
# Lint
# ###########

.PHONY: lint 
lint:
	${GO_BIN_PATH}/golangci-lint run

# ###########
# GoSec 
# ###########

.PHONY: gosec 
gosec:
	${GO_BIN_PATH}/gosec -exclude-dir=internal/repository ./...

# ###########
# Testing
# ###########

.PHONY: test-full
test-full:
	@go test -json -v -cover ./... 2>&1 | ${GO_BIN_PATH}/gotestfmt

.PHONY: test
test:
	@go test -json -v -short -cover ./... 2>&1 | ${GO_BIN_PATH}/gotestfmt

.PHONY: coverage
coverage:
	@go test -coverprofile=coverage.out ./... 2>&1 | ${GO_BIN_PATH}/gotestfmt
	@go tool cover -html=coverage.out

# ###########
# Migrations
# ###########

# Creates new migration up/down files in the 'migration' folder with the provided name.
.PHONY: migration/create
migration/create:
	@read -p "Enter migration name: " migration; \
	${GO_BIN_PATH}/migrate create -ext sql -dir ./gateways/repository/pg/migrations/ "$$migration"

# Drop migration.
.PHONY: migration/drop
migration/drop:
	dsn="postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):5432/$(DATABASE_NAME)?sslmode=disable&search_path=public"; \
	${GO_BIN_PATH}/migrate -source file://gateways/repository/pg/migrations -database $$dsn drop

# Execute the migrations up to the most recent one. Needs the following environment variables:
# DATABASE_HOST: database url
# DATABASE_USER: database user
# DATABASE_PASSWORD: database password
# DATABASE_NAME: database name
.PHONY: migration/up
migration/up:
	dsn="postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):5432/$(DATABASE_NAME)?sslmode=disable&search_path=public"; \
	${GO_BIN_PATH}/migrate -source file://gateways/repository/pg/migrations -database $$dsn up

# Rollback the migrations up to the oldest one. Needs the following environment variables:
# DATABASE_HOST: database url
# DATABASE_USER: database user
# DATABASE_PASSWORD: database password
# DATABASE_NAME: database name
.PHONY: migration/down
migration/down:
	dsn="postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):5432/$(DATABASE_NAME)?sslmode=disable&search_path=public"; \
	${GO_BIN_PATH}/migrate -source file://gateways/repository/pg/migrations -database $$dsn down
