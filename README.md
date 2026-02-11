# Flow Framework

> **Distributed Contract Testing & Tracing for Go Microservices**

Flow is a lightweight Go library that validates contracts between microservices at runtime. Service A defines what it **expects** to happen (Points), and Service B confirms what **actually** happened (Assertions). Flow compares them and reports any discrepancies.

```
┌─────────────┐                          ┌─────────────┐
│  Service A  │                          │  Service B  │
│  (Producer) │                          │  (Consumer) │
│             │   Start Flow             │             │
│   ┌─────────┤──────────────────────────┤─────────┐   │
│   │ Point 1 │  "order: {id, amount}"   │         │   │
│   │ Point 2 │  "payment: {status}"     │         │   │
│   └─────────┤                          │         │   │
│             │                          │ Assert 1│   │
│             │   GetFlow + Assert       │ Assert 2│   │
│             │◄─────────────────────────┤ Finish()│   │
└─────────────┘                          └─────────────┘
                         │
                    ┌────┴────┐
                    │ Compare │
                    │ Points  │
                    │   vs    │
                    │Assertions│
                    └────┬────┘
                         │
                  ✅ Match / ⚠️ Diff
```

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Usage Patterns](#usage-patterns)
- [Error Handling](#error-handling)
- [Logging](#logging)
- [Dashboard](#dashboard)
- [Project Structure](#project-structure)
- [Running Tests](#running-tests)

---

## Installation

### Prerequisites

- Go 1.24+
- PostgreSQL 15+ (or use Docker)

### Setup

```bash
# Clone the repository
git clone <repo-url> flow-tool
cd flow-tool

# Start PostgreSQL with Docker
make up

# Verify
go build ./...
go test ./pkg/flow/... -v
```

The database schema is applied **automatically** when `IsProduction` is `false`.

---

## Quick Start

### Service A — The Producer (defines expectations)

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "flow-tool/pkg/flow"
    _ "github.com/lib/pq"
)

func main() {
    db, _ := sql.Open("postgres",
        "user=user password=password dbname=flow_db sslmode=disable host=127.0.0.1 port=5432")
    defer db.Close()

    // Create client
    client, _ := flow.NewClient(db, flow.FlowConfig{
        ServiceName:   "order-service",
        IsProduction:  false,
        MaxExecutions: 10,
    })
    defer client.Close()

    ctx := context.Background()

    // Start a flow with a unique identifier
    f, _ := client.Start(ctx, "Create Order Flow", "ORD-001")

    // Define expectation points
    f.CreatePoint(ctx, "Order Created", map[string]interface{}{
        "id":     "ORD-001",
        "amount": 99.90,
        "status": "PENDING",
    })

    f.CreatePoint(ctx, "Payment Processed", map[string]interface{}{
        "status": "PAID",
        "method": "credit_card",
    })

    fmt.Println("✅ Flow started with 2 expectation points")
}
```

### Service B — The Consumer (validates reality)

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "flow-tool/pkg/flow"
    _ "github.com/lib/pq"
)

func main() {
    db, _ := sql.Open("postgres",
        "user=user password=password dbname=flow_db sslmode=disable host=127.0.0.1 port=5432")
    defer db.Close()

    client, _ := flow.NewClient(db, flow.FlowConfig{
        ServiceName:  "payment-service",
        IsProduction: false,
    })
    defer client.Close()

    ctx := context.Background()

    // Retrieve the active flow by name + identifier
    f, _ := client.GetFlow(ctx, "Create Order Flow", "ORD-001")

    // Add assertions (the ACTUAL values observed)
    f.AddAssertion(ctx, map[string]interface{}{
        "id":     "ORD-001",
        "amount": 99.90,
        "status": "PENDING",
    })

    f.AddAssertion(ctx, map[string]interface{}{
        "status": "PAID",
        "method": "credit_card",
    })

    // Finish and compare points vs assertions
    result, _ := f.Finish(ctx)

    if result.Success {
        fmt.Printf("✅ Contract valid! Execution time: %s\n", result.ExecutionTime)
    } else {
        fmt.Printf("⚠️  %d discrepancies found:\n", result.ErrorCount)
        for _, d := range result.Discrepancies {
            fmt.Printf("  - %s: %s\n", d.Description, d.Diff)
        }
    }
}
```

### Run the Example

```bash
# Terminal 1: Start database
make up

# Terminal 2: Run producer
make run-a

# Terminal 3: Run consumer
make run-b

# Terminal 4: View dashboard
make dashboard
# Open http://localhost:8585
```

---

## Core Concepts

| Concept | Description |
|---------|-------------|
| **Flow** | A tracked conversation between services. Identified by `name` + optional `identifier` |
| **Point** | An expectation defined by the producer: "this is what SHOULD happen" |
| **Assertion** | A confirmation from the consumer: "this is what ACTUALLY happened" |
| **Finish** | Compares Points vs Assertions in order. Returns all discrepancies |
| **FlowClient** | The main entry point. Creates and retrieves flows |
| **FlowExecutor** | A running flow instance. Creates points, adds assertions, finishes |

### Flow Lifecycle

```
Start() → CreatePoint()... → [other service] → GetFlow() → AddAssertion()... → Finish()
   │                                                                              │
   │  ← Points (expected) ─────────── compared with ──── Assertions (actual) →    │
   │                                                                              │
   └──────────────────────── FinishResult{Success, Discrepancies} ───────────────┘
```

---

## Configuration

### Using Builder Pattern (recommended)

```go
client, err := flow.NewClientBuilder().
    WithDB(db).
    WithServiceName("my-service").
    WithProductionMode(false).
    WithMaxExecutions(100).
    WithCaching(true, 500).
    WithConnectionPool(5, 25, 5*time.Minute).
    WithTimeout(30 * time.Second).
    WithLogger(flow.NewStdLogger()).
    Build()
```

### Using Direct Config

```go
client, err := flow.NewClient(db, flow.FlowConfig{
    ServiceName:   "my-service",
    IsProduction:  false,
    MaxExecutions: 100,
    CacheEnabled:  true,
    MaxCacheSize:  500,
    Timeout:       30 * time.Second,
})
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ServiceName` | `string` | `""` | Name of the service (stored with each point/assertion) |
| `IsProduction` | `bool` | `false` | If `true`, all operations are no-ops (zero overhead) |
| `MaxExecutions` | `int` | `0` | Max flows with the same name. `0` = unlimited |
| `CacheEnabled` | `bool` | `false` | Enable in-memory caching for active flows |
| `MaxCacheSize` | `int` | `1000` | Max number of cached flows |
| `Timeout` | `time.Duration` | `30s` | Default timeout for operations |
| `SchemaEnabled` | `bool` | `false` | Enable JSON schema validation |
| `BatchSize` | `int` | `100` | Batch size for bulk operations |

### Connection Pool

```go
.WithConnectionPool(
    5,              // MaxIdleConns
    25,             // MaxOpenConns
    5*time.Minute,  // ConnMaxLifetime
)
```

### YAML Configuration

```yaml
# flow.config.yaml
database:
  host: 127.0.0.1
  port: 5432
  user: user
  password: password
  name: flow_db

server:
  port: 8585
```

```go
cfg, _ := config.LoadConfig("flow.config.yaml")
db, _ := sql.Open("postgres", cfg.GetConnString())
```

---

## API Reference

### FlowClient

```go
// Create a new client
func NewClient(db *sql.DB, config FlowConfig) (*FlowClient, error)

// Start a new flow. Previous active flows with the same name/identifier are interrupted.
func (c *FlowClient) Start(ctx context.Context, flowName string, identifier ...string) (*flowInstance, error)

// Retrieve an existing active flow.
func (c *FlowClient) GetFlow(ctx context.Context, flowName string, identifier ...string) (*flowInstance, error)

// Release resources.
func (c *FlowClient) Close() error
```

### FlowExecutor (flow instance)

```go
// Create an expectation point.
func (f *flowInstance) CreatePoint(ctx context.Context, description string, expected interface{}, opts ...PointOption) error

// Record an actual observed value.
func (f *flowInstance) AddAssertion(ctx context.Context, actual interface{}) error

// Compare all points vs assertions and return the result.
func (f *flowInstance) Finish(ctx context.Context) (*FinishResult, error)

// Get flow metadata.
func (f *flowInstance) GetFlowInfo() *Flow
```

### Point Options

```go
// Attach a JSON schema for validation
f.CreatePoint(ctx, "Order", data,
    flow.WithSchema([]byte(`{"type":"object"}`)),
)

// Set a timeout for the point
f.CreatePoint(ctx, "Payment", data,
    flow.WithTimeout(10 * time.Second),
)

// Combine options
f.CreatePoint(ctx, "Shipping", data,
    flow.WithSchema(schema),
    flow.WithTimeout(30 * time.Second),
)
```

### FinishResult

```go
type FinishResult struct {
    Success       bool          // true if all points match their assertions
    Discrepancies []Discrepancy // list of differences found
    ExecutionTime time.Duration // time from Start() to Finish()
    ErrorCount    int           // total number of errors
}

type Discrepancy struct {
    PointID     int64       // ID of the expected point
    AssertionID int64       // ID of the actual assertion (0 if missing)
    Description string      // Point description
    Expected    interface{} // expected value
    Actual      interface{} // actual value
    Diff        string      // human-readable diff message
    Timestamp   time.Time
}
```

### Deep Comparison

```go
// Compare two JSON values — returns ALL diffs (not just the first)
diffs, equal := flow.DeepCompare(expectedJSON, actualJSON)

for _, d := range diffs {
    fmt.Printf("Path: %s — %s\n", d.Path, d.Message)
}

// Backward-compatible string output
msg, equal := flow.DeepCompareString(expectedJSON, actualJSON)
```

---

## Usage Patterns

### 1. Middleware Pattern

Intercept messages and automatically validate contracts:

```go
func FlowMiddleware(client *flow.FlowClient, next Handler) Handler {
    return func(ctx context.Context, msg Message) error {
        f, err := client.GetFlow(ctx, msg.FlowID)
        if err == nil {
            f.AddAssertion(ctx, msg.Payload)
        }
        return next(ctx, msg)
    }
}
```

### 2. Clean Architecture (Adapter Pattern)

Decouple business logic from the Flow Framework:

```go
// domain/order.go
type OrderObserver interface {
    OnOrderCreated(order Order)
}

// infra/flow_adapter.go
type FlowOrderObserver struct {
    client *flow.FlowClient
}

func (o *FlowOrderObserver) OnOrderCreated(order domain.Order) {
    ctx := context.Background()
    f, _ := o.client.Start(ctx, "Order Flow", order.ID)
    f.CreatePoint(ctx, "Order Created", map[string]interface{}{
        "id":     order.ID,
        "amount": order.Amount,
    })
}

// usecase/create_order.go
func (uc *CreateOrderUseCase) Execute(id string, amount float64) error {
    order := domain.Order{ID: id, Amount: amount}
    uc.repo.Save(order)
    uc.observer.OnOrderCreated(order) // observer is FlowOrderObserver
    return nil
}
```

### 3. Production Mode (Zero Overhead)

```go
client, _ := flow.NewClientBuilder().
    WithDB(db).
    WithProductionMode(true). // ← all flow operations become no-ops
    Build()

// These calls do nothing and return immediately:
f, _ := client.Start(ctx, "My Flow")
f.CreatePoint(ctx, "Step 1", data) // no-op
f.Finish(ctx)                       // returns {Success: true}
```

---

## Error Handling

Flow uses **structured errors** with sentinel values for programmatic handling:

```go
f, err := client.GetFlow(ctx, "My Flow", "ID-123")
if err != nil {
    if flow.IsNotFound(err) {
        // Flow doesn't exist or is not active
        log.Println("Flow not found, starting a new one...")
        f, _ = client.Start(ctx, "My Flow", "ID-123")
    } else {
        // Database or other error
        log.Fatalf("Unexpected error: %v", err)
    }
}
```

### Available Error Checks

| Function | Sentinel | When |
|----------|----------|------|
| `flow.IsNotFound(err)` | `ErrFlowNotFound` | No active flow with that name/identifier |
| `flow.IsSkipped(err)` | `ErrFlowSkipped` | Operation skipped (production mode) |
| `flow.IsLimitReached(err)` | `ErrLimitReached` | `MaxExecutions` limit was hit |

### FlowError Structure

Every error includes the operation name and flow name for debugging:

```
flow.Start [order-flow]: failed to create flow: connection refused
flow.GetFlow [order-flow]: flow: not found
flow.Finish [order-flow]: failed to fetch points: timeout
```

---

## Logging

Flow supports pluggable logging via the `Logger` interface:

```go
// Use the built-in standard logger
client, _ := flow.NewClientBuilder().
    WithDB(db).
    WithLogger(flow.NewStdLogger()).
    Build()

// Use the fmt logger (prints to stdout)
client, _ := flow.NewClientBuilder().
    WithDB(db).
    WithLogger(flow.NewFmtLogger()).
    Build()
```

### Custom Logger

Implement the `Logger` interface to integrate with your logging library (e.g., zap, logrus, slog):

```go
type Logger interface {
    Debug(msg string, args ...interface{})
    Info(msg string, args ...interface{})
    Error(msg string, args ...interface{})
}

// Example: Zap adapter
type zapLogger struct {
    logger *zap.SugaredLogger
}

func (z *zapLogger) Debug(msg string, args ...interface{}) {
    z.logger.Debugf(msg, args...)
}
func (z *zapLogger) Info(msg string, args ...interface{}) {
    z.logger.Infof(msg, args...)
}
func (z *zapLogger) Error(msg string, args ...interface{}) {
    z.logger.Errorf(msg, args...)
}
```

**Default**: `noopLogger` (zero overhead — no logging at all).

---

## Dashboard

Flow includes a web dashboard to visualize flows, points, and assertions:

```bash
make dashboard
# Open http://localhost:8585
```

Features:
- List all flows with status (ACTIVE / FINISHED / INTERRUPTED)
- Timeline view with points and assertions side by side
- Compare expected vs actual values
- Search and filter flows
- Pagination with infinite scroll

---

## Project Structure

```
flow-tool/
├── pkg/flow/               # Core library
│   ├── flow.go             # FlowClient + flowInstance (business logic)
│   ├── storage.go          # PostgreSQL storage layer (all SQL)
│   ├── cache.go            # In-memory cache (thread-safe)
│   ├── types.go            # Data types (Flow, Point, Assertion, etc.)
│   ├── interfaces.go       # Interfaces (FlowTracker, FlowExecutor, Storage)
│   ├── builder.go          # ClientBuilder (fluent configuration)
│   ├── comparator.go       # Deep comparison engine (multi-diff)
│   ├── validation.go       # Schema validation
│   ├── errors.go           # Structured error types
│   ├── logger.go           # Logger interface + implementations
│   ├── flow_test.go        # Tests: cache, errors, builder, options
│   └── comparator_test.go  # Tests: deep comparison
│
├── pkg/config/             # YAML config loader
│   └── config.go
│
├── cmd/
│   ├── service-a/main.go   # Example: producer service
│   ├── service-b/main.go   # Example: consumer service
│   └── dashboard/          # Web dashboard
│       ├── main.go
│       └── static/         # HTML, CSS, JS
│
├── examples/
│   ├── decoupled/          # Decoupled example
│   ├── enhanced/           # Enhanced features example
│   ├── clean_architecture/ # Clean arch with adapters
│   └── middleware/         # Middleware pattern
│
├── docker-compose.yml      # PostgreSQL + pgAdmin
├── init.sql                # Database schema
├── flow.config.yaml        # Configuration
├── Makefile                # Dev commands
└── go.mod
```

---

## Running Tests

```bash
# Run all tests
go test ./pkg/flow/... -v

# Run with coverage
go test ./pkg/flow/... -cover

# Run specific test
go test ./pkg/flow/... -run TestDeepCompare -v

# Vet the code
go vet ./...
```

---

## License

MIT
