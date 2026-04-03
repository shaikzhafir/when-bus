when-bus is a middleman to the LTA public api for my own use

## Using the API

All endpoints are **GET** requests. Query parameters carry the “payload”; there is no request body.

Set **`LTA_API_KEY`** in the environment to your [LTA DataMall](https://datamall.lta.gov.sg/content/dam/datamall/datasets/LTA_DataMall_Specs.pdf) account key before starting the server. Without it, arrival and bus-stop calls will fail against LTA.

**Base URL**

- Local: `http://localhost:8080`
- Replace below with your own host if deployed (for example `https://example.com`).

### `GET /getBusArrival`

Returns live arrivals for one bus stop. **`busStopCode` is required** (LTA bus stop code, e.g. `71119`).

```bash
curl -sS 'http://localhost:8080/getBusArrival?busStopCode=71119'
```

Pretty-print with `jq`:

```bash
curl -sS 'http://localhost:8080/getBusArrival?busStopCode=71119' | jq .
```

**200 response** (JSON array of services):

```json
[
  {
    "ServiceNo": "63",
    "Operator": "SBST",
    "NextBuses": ["5", "15"],
    "LoadStatus": ["SEA", "SEA"],
    "IsWheelchair": true
  }
]
```

**400** if a required query parameter is missing or invalid (JSON body):

```bash
curl -sS -w '\nHTTP %{http_code}\n' 'http://localhost:8080/getBusArrival'
```

Example body:

```json
{
  "success": false,
  "error": "Query argument busStopCode is required, but not found",
  "reason": "missing_required_query_param:busStopCode"
}
```

### `GET /getNearestBusStops`

Returns the **nearest four** bus stops to a point, each with arrival data. **`lat`** and **`lng`** are required (WGS84 decimals).

```bash
curl -sS 'http://localhost:8080/getNearestBusStops?lat=1.326277&lng=103.890342'
```

With `jq`:

```bash
curl -sS 'http://localhost:8080/getNearestBusStops?lat=1.326277&lng=103.890342' | jq .
```

**200 response** (JSON array; `Distance` is kilometers):

```json
[
  {
    "BusStopCode": "71119",
    "RoadName": "…",
    "Description": "…",
    "Distance": 0.15,
    "Arrivals": [
      {
        "ServiceNo": "63",
        "Operator": "SBST",
        "NextBuses": ["5"],
        "LoadStatus": ["SEA"],
        "IsWheelchair": true
      }
    ]
  }
]
```

### `GET /api`

Optional demo endpoint (plain text). Query parameter **`name`** is optional.

```bash
curl -sS 'http://localhost:8080/api'
curl -sS 'http://localhost:8080/api?name=Alice'
```

### Errors

- **400** — OpenAPI validation (missing or malformed query params). Body is JSON with `success`, `error`, and `reason`.
- **500** — Upstream LTA failure, JSON parse error, or similar. Body is JSON with `success`, `error`, and `reason` (for example `upstream_or_parse_error`).

If you terminate TLS or add an edge worker in front of this service, responses may be wrapped or transformed; the shapes above describe the Go server directly.

The full contract is also defined in **`openapi.yaml`**.

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