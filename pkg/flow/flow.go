package flow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

const schema = `
CREATE TABLE IF NOT EXISTS flows (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    identifier VARCHAR(255),
    status VARCHAR(50) DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS points (
    id BIGSERIAL PRIMARY KEY,
    flow_id BIGINT REFERENCES flows(id) ON DELETE CASCADE,
    description TEXT,
    expected JSONB,
    service_name VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS assertions (
    id BIGSERIAL PRIMARY KEY,
    flow_id BIGINT REFERENCES flows(id) ON DELETE CASCADE,
    actual JSONB,
    service_name VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_points_flow_id ON points(flow_id);
CREATE INDEX IF NOT EXISTS idx_assertions_flow_id ON assertions(flow_id);
CREATE INDEX IF NOT EXISTS idx_flows_name_status ON flows(name, status);
CREATE INDEX IF NOT EXISTS idx_flows_identifier ON flows(identifier);
`

func NewClient(db *sql.DB, config FlowConfig) (*FlowClient, error) {
	if !config.IsProduction {
		if _, err := db.Exec(schema); err != nil {
			return nil, fmt.Errorf("failed to apply schema: %v", err)
		}
	}
	return &FlowClient{
		DB:     db,
		Config: config,
	}, nil
}

func (c *FlowClient) Start(flowName string, identifier ...string) (*FlowInstance, error) {
	if c.Config.IsProduction {
		return &FlowInstance{client: c, Flow: &Flow{Name: flowName, Status: "SKIPPED"}}, nil
	}

	ident := ""
	if len(identifier) > 0 {
		ident = identifier[0]
	}

	if c.Config.MaxExecutions > 0 {
		var count int
		// Check global count for this Flow Name
		err := c.DB.QueryRow("SELECT COUNT(*) FROM flows WHERE name = $1", flowName).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to count flows: %v", err)
		}
		if count >= c.Config.MaxExecutions {
			return &FlowInstance{client: c, Flow: &Flow{Name: flowName, Status: "SKIPPED_LIMIT"}}, nil
		}
	}

	// Interrupt existing ACTIVE flow with same Name & Identifier
	query := "UPDATE flows SET status = 'INTERRUPTED' WHERE name = $1 AND status = 'ACTIVE'"
	args := []interface{}{flowName}
	if ident != "" {
		query += " AND identifier = $2"
		args = append(args, ident)
	} else {
		query += " AND identifier IS NULL"
	}

	_, err := c.DB.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to interrupt existing flow: %v", err)
	}

	var id int64
	insertQuery := "INSERT INTO flows (name, identifier, status) VALUES ($1, $2, 'ACTIVE') RETURNING id"
	var identArg interface{} = ident
	if ident == "" {
		identArg = nil
	}

	err = c.DB.QueryRow(insertQuery, flowName, identArg).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create flow: %v", err)
	}

	return &FlowInstance{
		client: c,
		Flow:   &Flow{ID: id, Name: flowName, Identifier: ident, Status: "ACTIVE"},
	}, nil
}

func (c *FlowClient) GetFlow(flowName string, identifier ...string) (*FlowInstance, error) {
	if c.Config.IsProduction {
		return &FlowInstance{client: c, Flow: &Flow{Name: flowName, Status: "SKIPPED"}}, nil
	}

	ident := ""
	if len(identifier) > 0 {
		ident = identifier[0]
	}

	var f Flow
	query := "SELECT id, name, identifier, status, created_at FROM flows WHERE name = $1 AND status = 'ACTIVE'"
	args := []interface{}{flowName}

	if ident != "" {
		query += " AND identifier = $2"
		args = append(args, ident)
	} else {
		query += " AND identifier IS NULL"
	}
	query += " ORDER BY id DESC LIMIT 1"

	var identSql sql.NullString
	err := c.DB.QueryRow(query, args...).Scan(
		&f.ID, &f.Name, &identSql, &f.Status, &f.CreatedAt,
	)
	f.Identifier = identSql.String

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active flow found with name '%s' and identifier '%s'", flowName, ident)
		}
		return nil, fmt.Errorf("error fetching flow: %v", err)
	}
	return &FlowInstance{client: c, Flow: &f}, nil
}

func (f *FlowInstance) CreatePoint(description string, expected interface{}) error {
	if f.client.Config.IsProduction || (len(f.Flow.Status) >= 7 && f.Flow.Status[:7] == "SKIPPED") {
		return nil
	}

	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return fmt.Errorf("failed to marshal expected value: %v", err)
	}

	_, err = f.client.DB.Exec("INSERT INTO points (flow_id, description, expected, service_name) VALUES ($1, $2, $3, $4)",
		f.Flow.ID, description, expectedJSON, f.client.Config.ServiceName)

	if err != nil {
		return fmt.Errorf("failed to create point: %v", err)
	}
	return nil
}

func (f *FlowInstance) AddAssertion(actual interface{}) error {
	if f.client.Config.IsProduction || (len(f.Flow.Status) >= 7 && f.Flow.Status[:7] == "SKIPPED") {
		return nil
	}

	actualJSON, err := json.Marshal(actual)
	if err != nil {
		return fmt.Errorf("failed to marshal actual value: %v", err)
	}

	_, err = f.client.DB.Exec("INSERT INTO assertions (flow_id, actual, service_name) VALUES ($1, $2, $3)",
		f.Flow.ID, actualJSON, f.client.Config.ServiceName)

	if err != nil {
		return fmt.Errorf("failed to add assertion: %v", err)
	}
	return nil
}

func (f *FlowInstance) Finish() (*FinishResult, error) {
	if f.client.Config.IsProduction {
		return &FinishResult{Success: true}, nil
	}

	_, err := f.client.DB.Exec("UPDATE flows SET status = 'FINISHED' WHERE id = $1", f.Flow.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to close flow: %v", err)
	}

	return f.executeWorker()
}

type mixedEvent struct {
	Type      string
	Timestamp time.Time
	Point     *Point
	Assertion *Assertion
}

func (f *FlowInstance) executeWorker() (*FinishResult, error) {
	pRows, err := f.client.DB.Query("SELECT id, description, expected, created_at FROM points WHERE flow_id = $1 ORDER BY created_at ASC", f.Flow.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch points: %v", err)
	}
	defer pRows.Close()

	var points []Point
	for pRows.Next() {
		var p Point
		var expectedBytes []byte
		if err := pRows.Scan(&p.ID, &p.Description, &expectedBytes, &p.CreatedAt); err != nil {
			return nil, err
		}
		if expectedBytes != nil {
			p.Expected = json.RawMessage(expectedBytes)
		}
		points = append(points, p)
	}

	aRows, err := f.client.DB.Query("SELECT id, actual, created_at FROM assertions WHERE flow_id = $1 ORDER BY created_at ASC", f.Flow.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assertions: %v", err)
	}
	defer aRows.Close()

	var assertions []Assertion
	for aRows.Next() {
		var a Assertion
		var actualBytes []byte
		if err := aRows.Scan(&a.ID, &actualBytes, &a.CreatedAt); err != nil {
			return nil, err
		}
		if actualBytes != nil {
			a.Actual = json.RawMessage(actualBytes)
		}
		assertions = append(assertions, a)
	}

	var discrepancies []Discrepancy

	maxLen := len(points)
	if len(assertions) > maxLen {
		maxLen = len(assertions)
	}

	for i := 0; i < maxLen; i++ {
		if i >= len(assertions) {
			discrepancies = append(discrepancies, Discrepancy{
				PointID:     points[i].ID,
				Description: points[i].Description,
				Diff:        "Missing assertion for this point",
			})
			continue
		}

		if i >= len(points) {
			discrepancies = append(discrepancies, Discrepancy{
				AssertionID: assertions[i].ID,
				Description: "Orphan Assertion",
				Diff:        fmt.Sprintf("Assertion #%d found without a matching Point #%d", i+1, i+1),
			})
			continue
		}

		p := points[i]
		a := assertions[i]

		diff, equal := DeepCompare(p.Expected, a.Actual)
		if !equal {
			var expectedVal, actualVal interface{}
			_ = json.Unmarshal(p.Expected, &expectedVal)
			_ = json.Unmarshal(a.Actual, &actualVal)

			discrepancies = append(discrepancies, Discrepancy{
				PointID:     p.ID,
				AssertionID: a.ID,
				Description: p.Description,
				Expected:    expectedVal,
				Actual:      actualVal,
				Diff:        diff,
			})
		}
	}

	return &FinishResult{
		Success:       len(discrepancies) == 0,
		Discrepancies: discrepancies,
	}, nil
}
