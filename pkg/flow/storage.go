package flow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

const schemaDDL = `
CREATE TABLE IF NOT EXISTS flows (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    identifier VARCHAR(255),
    status VARCHAR(50) DEFAULT 'ACTIVE',
    service VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS points (
    id BIGSERIAL PRIMARY KEY,
    flow_id BIGINT REFERENCES flows(id) ON DELETE CASCADE,
    description TEXT,
    expected JSONB,
    service_name VARCHAR(255),
    schema JSONB,
    timeout BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS assertions (
    id BIGSERIAL PRIMARY KEY,
    flow_id BIGINT REFERENCES flows(id) ON DELETE CASCADE,
    actual JSONB,
    service_name VARCHAR(255),
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_points_flow_id ON points(flow_id);
CREATE INDEX IF NOT EXISTS idx_assertions_flow_id ON assertions(flow_id);
CREATE INDEX IF NOT EXISTS idx_flows_name_status ON flows(name, status);
CREATE INDEX IF NOT EXISTS idx_flows_identifier ON flows(identifier);
`

// pgStorage implements Storage using PostgreSQL.
type pgStorage struct {
	db *sql.DB
}

func newPGStorage(db *sql.DB) *pgStorage {
	return &pgStorage{db: db}
}

func (s *pgStorage) ApplySchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, schemaDDL)
	if err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}
	return nil
}

func (s *pgStorage) CountFlowsByName(ctx context.Context, flowName string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM flows WHERE name = $1", flowName).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count flows: %w", err)
	}
	return count, nil
}

func (s *pgStorage) InterruptActiveFlows(ctx context.Context, flowName, identifier string) error {
	query := "UPDATE flows SET status = 'INTERRUPTED', updated_at = CURRENT_TIMESTAMP WHERE name = $1 AND status = 'ACTIVE'"
	args := []interface{}{flowName}
	if identifier != "" {
		query += " AND identifier = $2"
		args = append(args, identifier)
	} else {
		query += " AND identifier IS NULL"
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to interrupt flows: %w", err)
	}
	return nil
}

func (s *pgStorage) InsertFlow(ctx context.Context, flowName, identifier, serviceName string) (int64, error) {
	var identArg interface{} = identifier
	if identifier == "" {
		identArg = nil
	}

	var id int64
	err := s.db.QueryRowContext(ctx,
		"INSERT INTO flows (name, identifier, status, service) VALUES ($1, $2, 'ACTIVE', $3) RETURNING id",
		flowName, identArg, serviceName,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create flow: %w", err)
	}
	return id, nil
}

func (s *pgStorage) FindActiveFlow(ctx context.Context, flowName, identifier string) (*Flow, error) {
	query := "SELECT id, name, identifier, status, created_at FROM flows WHERE name = $1 AND status = 'ACTIVE'"
	args := []interface{}{flowName}

	if identifier != "" {
		query += " AND identifier = $2"
		args = append(args, identifier)
	} else {
		query += " AND identifier IS NULL"
	}
	query += " ORDER BY id DESC LIMIT 1"

	var f Flow
	var identSql sql.NullString
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&f.ID, &f.Name, &identSql, &f.Status, &f.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &FlowError{
				Op:       "GetFlow",
				FlowName: flowName,
				Err:      ErrFlowNotFound,
			}
		}
		return nil, fmt.Errorf("error fetching flow: %w", err)
	}
	f.Identifier = identSql.String
	return &f, nil
}

func (s *pgStorage) FinishFlow(ctx context.Context, flowID int64) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE flows SET status = 'FINISHED', updated_at = CURRENT_TIMESTAMP WHERE id = $1", flowID)
	if err != nil {
		return fmt.Errorf("failed to finish flow: %w", err)
	}
	return nil
}

func (s *pgStorage) InsertPoint(ctx context.Context, p *Point) error {
	var schemaArg interface{}
	if p.Schema != nil {
		schemaArg = []byte(p.Schema)
	}
	var timeoutArg interface{}
	if p.Timeout != nil {
		timeoutArg = p.Timeout.Milliseconds()
	}

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO points (flow_id, description, expected, service_name, schema, timeout) VALUES ($1, $2, $3, $4, $5, $6)",
		p.FlowID, p.Description, []byte(p.Expected), p.ServiceName, schemaArg, timeoutArg)
	if err != nil {
		return fmt.Errorf("failed to create point: %w", err)
	}
	return nil
}

func (s *pgStorage) InsertAssertion(ctx context.Context, flowID int64, actualJSON []byte, serviceName string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO assertions (flow_id, actual, service_name, processed_at) VALUES ($1, $2, $3, $4)",
		flowID, actualJSON, serviceName, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add assertion: %w", err)
	}
	return nil
}

func (s *pgStorage) FetchPointsAndAssertions(ctx context.Context, flowID int64) ([]Point, []Assertion, error) {
	type pointsResult struct {
		points []Point
		err    error
	}
	type assertionsResult struct {
		assertions []Assertion
		err        error
	}

	pCh := make(chan pointsResult, 1)
	aCh := make(chan assertionsResult, 1)

	go func() {
		points, err := s.fetchPoints(ctx, flowID)
		pCh <- pointsResult{points, err}
	}()

	go func() {
		assertions, err := s.fetchAssertions(ctx, flowID)
		aCh <- assertionsResult{assertions, err}
	}()

	pRes := <-pCh
	if pRes.err != nil {
		return nil, nil, pRes.err
	}

	aRes := <-aCh
	if aRes.err != nil {
		return nil, nil, aRes.err
	}

	return pRes.points, aRes.assertions, nil
}

func (s *pgStorage) fetchPoints(ctx context.Context, flowID int64) ([]Point, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, description, expected, created_at FROM points WHERE flow_id = $1 ORDER BY created_at ASC", flowID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch points: %w", err)
	}
	defer rows.Close()

	var points []Point
	for rows.Next() {
		var p Point
		var expectedBytes []byte
		if err := rows.Scan(&p.ID, &p.Description, &expectedBytes, &p.CreatedAt); err != nil {
			return nil, err
		}
		if expectedBytes != nil {
			p.Expected = json.RawMessage(expectedBytes)
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (s *pgStorage) fetchAssertions(ctx context.Context, flowID int64) ([]Assertion, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, actual, created_at FROM assertions WHERE flow_id = $1 ORDER BY created_at ASC", flowID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assertions: %w", err)
	}
	defer rows.Close()

	var assertions []Assertion
	for rows.Next() {
		var a Assertion
		var actualBytes []byte
		if err := rows.Scan(&a.ID, &actualBytes, &a.CreatedAt); err != nil {
			return nil, err
		}
		if actualBytes != nil {
			a.Actual = json.RawMessage(actualBytes)
		}
		assertions = append(assertions, a)
	}
	return assertions, rows.Err()
}
