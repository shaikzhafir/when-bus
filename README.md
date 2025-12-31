when-bus is a middleman to the LTA public api for my own use

## OpenAPI Code Generation

This project uses OpenAPI specification for API documentation and code generation. The generated code uses standard library `net/http` only - no external frameworks.

### Files
- `openapi.yaml` - OpenAPI 3.0 specification defining the API endpoints
- `internal/generated/api.gen.go` - Generated server code (do not edit manually)

### Generating Server Code

To generate Go server code from the OpenAPI specification:

**Using Make:**
```bash
make generate-api
```

**Using the script:**
```bash
./generate-api.sh
```

**Manual command:**
```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
oapi-codegen -generate types,std-http -package generated -o internal/generated/api.gen.go openapi.yaml
```

The generated code will be placed in `internal/generated/api.gen.go` and uses standard library `net/http` handlers.

### Other Make Targets
- `make install-tools` - Install oapi-codegen tool
- `make validate-openapi` - Validate the OpenAPI specification
- `make help` - Show all available make targets 