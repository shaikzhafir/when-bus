#!/bin/bash

# Script to generate Go server code from OpenAPI specification

set -e

echo "Generating API code from OpenAPI spec..."

# Check if oapi-codegen is installed
if ! command -v oapi-codegen &> /dev/null; then
    echo "oapi-codegen not found. Installing..."
    go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest
fi

# Create generated directory if it doesn't exist
mkdir -p internal/generated

# Generate the code
oapi-codegen -generate types,server -package generated -o internal/generated/api.gen.go openapi.yaml

echo "API code generated successfully in internal/generated/api.gen.go"

