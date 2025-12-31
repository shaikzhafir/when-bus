im just using this so i dont have to rebuild this shit everytime i have some sideproject idea that i will never complete and leave it hanging

- uses tailwindcss
- uses go templating + htmx for client interaction with server
- all built into a single container

dependencies
1) tailwindcss, https://tailwindcss.com/docs/installation , follow standalone executable
   - note: if u are adding new folder for html files, please update tailwind.config.js to apply those changes
2) entr for hot reload, https://jvns.ca/blog/2020/06/28/entr/
3) htmx for simple client server interactivity

## OpenAPI Code Generation

This project includes OpenAPI specification for API documentation and code generation.

### Files
- `openapi.yaml` - OpenAPI 3.0 specification defining the API endpoints

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
go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest
oapi-codegen -generate types,server -package generated -o internal/generated/api.gen.go openapi.yaml
```

The generated code will be placed in `internal/generated/api.gen.go`.

### Other Make Targets
- `make install-tools` - Install oapi-codegen tool
- `make validate-openapi` - Validate the OpenAPI specification
- `make help` - Show all available make targets 