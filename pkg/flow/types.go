package flow

import (
	"encoding/json"
	"time"
)

type Flow struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	Identifier string          `json:"identifier,omitempty"`
	Status     string          `json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	Service    string          `json:"service"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

type Point struct {
	ID          int64           `json:"id"`
	FlowID      int64           `json:"flow_id"`
	Description string          `json:"description"`
	Expected    json.RawMessage `json:"expected"`
	ServiceName string          `json:"service_name"`
	CreatedAt   time.Time       `json:"created_at"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	Timeout     *time.Duration  `json:"timeout,omitempty"`
}

type Assertion struct {
	ID          int64           `json:"id"`
	FlowID      int64           `json:"flow_id"`
	Actual      json.RawMessage `json:"actual"`
	ServiceName string          `json:"service_name"`
	CreatedAt   time.Time       `json:"created_at"`
	ProcessedAt *time.Time      `json:"processed_at,omitempty"`
}

type FinishResult struct {
	Success       bool          `json:"success"`
	Discrepancies []Discrepancy `json:"discrepancies,omitempty"`
	ExecutionTime time.Duration `json:"execution_time"`
	ErrorCount    int           `json:"error_count"`
}

type Discrepancy struct {
	PointID     int64       `json:"point_id"`
	AssertionID int64       `json:"assertion_id,omitempty"`
	Description string      `json:"description"`
	Expected    interface{} `json:"expected"`
	Actual      interface{} `json:"actual"`
	Diff        string      `json:"diff"`
	Timestamp   time.Time   `json:"timestamp"`
}

type SchemaValidator struct {
	Schema json.RawMessage `json:"schema"`
}

type PointOption func(*Point)

func WithSchema(schema json.RawMessage) PointOption {
	return func(p *Point) {
		p.Schema = schema
	}
}

func WithTimeout(d time.Duration) PointOption {
	return func(p *Point) {
		p.Timeout = &d
	}
}
