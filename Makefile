.PHONY: generate-api help

# OpenAPI code generation
generate-api:
	@echo "Generating API code from OpenAPI spec..."
	@which oapi-codegen > /dev/null || (echo "oapi-codegen not found. Installing..." && go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest)
	@mkdir -p internal/generated
	oapi-codegen -generate types,server -package generated -o internal/generated/api.gen.go openapi.yaml
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

help:
	@echo "Available targets:"
	@echo "  make generate-api      - Generate Go server code from openapi.yaml"
	@echo "  make install-tools     - Install oapi-codegen tool"
	@echo "  make validate-openapi  - Validate the OpenAPI specification"
	@echo "  make help              - Show this help message"

