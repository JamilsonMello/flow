package flow

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Flow struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Point struct {
	ID          int64           `json:"id"`
	FlowID      int64           `json:"flow_id"`
	Description string          `json:"description"`
	Expected    json.RawMessage `json:"expected"`
	ServiceName string          `json:"service_name"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Assertion struct {
	ID          int64           `json:"id"`
	FlowID      int64           `json:"flow_id"`
	Actual      json.RawMessage `json:"actual"`
	ServiceName string          `json:"service_name"`
	CreatedAt   time.Time       `json:"created_at"`
}

type FlowConfig struct {
	ServiceName   string
	IsProduction  bool
	MaxExecutions int
}

type FlowClient struct {
	DB     *sql.DB
	Config FlowConfig
}

type FlowInstance struct {
	client *FlowClient
	Flow   *Flow
}

type FinishResult struct {
	Success       bool          `json:"success"`
	Discrepancies []Discrepancy `json:"discrepancies,omitempty"`
}

type Discrepancy struct {
	PointID     int64       `json:"point_id"`
	AssertionID int64       `json:"assertion_id,omitempty"`
	Description string      `json:"description"`
	Expected    interface{} `json:"expected"`
	Actual      interface{} `json:"actual"`
	Diff        string      `json:"diff"`
}
