.PHONY: generate-api help install-tools validate-openapi run run-hot run-plain build test

BINARY := when-bus

# OpenAPI code generation
generate-api:
	@echo "Generating API code from OpenAPI spec..."
	@which oapi-codegen > /dev/null || (echo "oapi-codegen not found. Installing..." && go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest)
	@mkdir -p internal/generated
	oapi-codegen -generate types,std-http -package generated -o internal/generated/api.gen.go openapi.yaml
	@echo "API code generated successfully!"

# Install oapi-codegen if not present
install-tools:
	@echo "Installing oapi-codegen..."
	go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest
	@echo "Tools installed successfully!"

# Validate OpenAPI spec
validate-openapi:
	@which oapi-codegen > /dev/null || (echo "oapi-codegen not found. Run 'make install-tools' first" && exit 1)
	@echo "Validating OpenAPI spec..."
	oapi-codegen -generate types -package generated -o /dev/null openapi.yaml
	@echo "OpenAPI spec is valid!"

# Run the HTTP server locally with hot reload (entr + reload.sh; brew install entr).
# Loads .env inside reload.sh. Use run-plain for a single run without entr.
run run-hot:
	@bash reload.sh

# One-shot run (no file watching). Sources .env if present.
run-plain:
	set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/server

# Build a binary to bin/$(BINARY)
build:
	@mkdir -p bin
	go build -o bin/$(BINARY) ./cmd/server

test:
	go test ./...

help:
	@echo "Available targets:"
	@echo "  make run               - Run the server with hot reload (entr; sources .env)"
	@echo "  make run-hot           - Alias for make run"
	@echo "  make run-plain         - Run once without entr (sources .env)"
	@echo "  make build             - Build bin/$(BINARY)"
	@echo "  make test              - Run all tests"
	@echo "  make generate-api      - Generate Go server code from openapi.yaml"
	@echo "  make install-tools     - Install oapi-codegen tool"
	@echo "  make validate-openapi  - Validate the OpenAPI specification"
	@echo "  make help              - Show this help message"

