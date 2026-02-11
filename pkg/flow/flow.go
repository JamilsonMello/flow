package flow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type FlowClient struct {
	DB      *sql.DB
	Config  FlowConfig
	storage *pgStorage
	cache   *flowCache
	logger  Logger
}

type flowInstance struct {
	client    *FlowClient
	Flow      *Flow
	startTime time.Time
}

func NewClient(db *sql.DB, config FlowConfig) (*FlowClient, error) {
	client := &FlowClient{
		DB:      db,
		Config:  config,
		storage: newPGStorage(db),
		cache:   newFlowCache(config.CacheEnabled, config.MaxCacheSize),
		logger:  noopLogger{},
	}

	if !config.IsProduction {
		if err := client.storage.ApplySchema(context.Background()); err != nil {
			return nil, err
		}
	}

	return client, nil
}

func (c *FlowClient) Close() error {
	c.cache.Clear()
	c.logger.Info("FlowClient closed")
	return nil
}

func isSkipped(status string) bool {
	return len(status) >= 7 && status[:7] == "SKIPPED"
}

func (c *FlowClient) Start(ctx context.Context, flowName string, identifier ...string) (*flowInstance, error) {
	if c.Config.IsProduction {
		c.logger.Debug("Production mode: skipping flow '%s'", flowName)
		return &flowInstance{client: c, Flow: &Flow{Name: flowName, Status: "SKIPPED"}, startTime: time.Now()}, nil
	}

	ident := ""
	if len(identifier) > 0 {
		ident = identifier[0]
	}

	if c.Config.MaxExecutions > 0 {
		count, err := c.storage.CountFlowsByName(ctx, flowName)
		if err != nil {
			return nil, &FlowError{Op: "Start", FlowName: flowName, Err: err}
		}
		if count >= c.Config.MaxExecutions {
			c.logger.Info("Limit reached for flow '%s' (%d/%d)", flowName, count, c.Config.MaxExecutions)
			return &flowInstance{client: c, Flow: &Flow{Name: flowName, Status: "SKIPPED_LIMIT"}, startTime: time.Now()}, nil
		}
	}

	if err := c.storage.InterruptActiveFlows(ctx, flowName, ident); err != nil {
		return nil, &FlowError{Op: "Start", FlowName: flowName, Err: err}
	}
	c.cache.Delete(flowName, ident)

	id, err := c.storage.InsertFlow(ctx, flowName, ident, c.Config.ServiceName)
	if err != nil {
		return nil, &FlowError{Op: "Start", FlowName: flowName, Err: err}
	}

	f := &Flow{ID: id, Name: flowName, Identifier: ident, Status: "ACTIVE", Service: c.Config.ServiceName}
	c.cache.Set(flowName, ident, f)
	c.logger.Info("Flow started: '%s' (id=%d)", flowName, id)

	return &flowInstance{
		client:    c,
		Flow:      f,
		startTime: time.Now(),
	}, nil
}

func (c *FlowClient) GetFlow(ctx context.Context, flowName string, identifier ...string) (*flowInstance, error) {
	if c.Config.IsProduction {
		return &flowInstance{client: c, Flow: &Flow{Name: flowName, Status: "SKIPPED"}, startTime: time.Now()}, nil
	}

	ident := ""
	if len(identifier) > 0 {
		ident = identifier[0]
	}

	if cached, ok := c.cache.Get(flowName, ident); ok {
		c.logger.Debug("Cache hit for flow '%s'", flowName)
		return &flowInstance{client: c, Flow: cached, startTime: time.Now()}, nil
	}

	f, err := c.storage.FindActiveFlow(ctx, flowName, ident)
	if err != nil {
		if IsNotFound(err) && c.Config.MaxExecutions > 0 {
			count, countErr := c.storage.CountFlowsByName(ctx, flowName)
			if countErr == nil && count >= c.Config.MaxExecutions {
				c.logger.Info("GetFlow: flow '%s' reached execution limit (%d/%d), skipping", flowName, count, c.Config.MaxExecutions)
				return &flowInstance{client: c, Flow: &Flow{Name: flowName, Status: "SKIPPED_LIMIT"}, startTime: time.Now()}, nil
			}
		}
		return nil, err
	}

	c.cache.Set(flowName, ident, f)
	return &flowInstance{client: c, Flow: f, startTime: time.Now()}, nil
}

func (f *flowInstance) GetFlowInfo() *Flow {
	return f.Flow
}

func (f *flowInstance) CreatePoint(ctx context.Context, description string, expected interface{}, opts ...PointOption) error {
	if f.client.Config.IsProduction || isSkipped(f.Flow.Status) {
		return nil
	}

	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return fmt.Errorf("failed to marshal expected value: %w", err)
	}

	p := &Point{
		FlowID:      f.Flow.ID,
		Description: description,
		Expected:    expectedJSON,
		ServiceName: f.client.Config.ServiceName,
	}

	for _, opt := range opts {
		opt(p)
	}

	if err := f.client.storage.InsertPoint(ctx, p); err != nil {
		return &FlowError{Op: "CreatePoint", FlowName: f.Flow.Name, Err: err}
	}

	f.client.logger.Debug("Point created: '%s' on flow '%s'", description, f.Flow.Name)
	return nil
}

func (f *flowInstance) AddAssertion(ctx context.Context, actual interface{}) error {
	if f.client.Config.IsProduction || isSkipped(f.Flow.Status) {
		return nil
	}

	actualJSON, err := json.Marshal(actual)
	if err != nil {
		return fmt.Errorf("failed to marshal actual value: %w", err)
	}

	if err := f.client.storage.InsertAssertion(ctx, f.Flow.ID, actualJSON, f.client.Config.ServiceName); err != nil {
		return &FlowError{Op: "AddAssertion", FlowName: f.Flow.Name, Err: err}
	}

	f.client.logger.Debug("Assertion added to flow '%s'", f.Flow.Name)
	return nil
}

func (f *flowInstance) Finish(ctx context.Context) (*FinishResult, error) {
	if f.client.Config.IsProduction || isSkipped(f.Flow.Status) {
		return &FinishResult{Success: true}, nil
	}

	if err := f.client.storage.FinishFlow(ctx, f.Flow.ID); err != nil {
		return nil, &FlowError{Op: "Finish", FlowName: f.Flow.Name, Err: err}
	}

	f.client.cache.Delete(f.Flow.Name, f.Flow.Identifier)
	return f.executeWorker(ctx)
}

func (f *flowInstance) executeWorker(ctx context.Context) (*FinishResult, error) {
	points, assertions, err := f.client.storage.FetchPointsAndAssertions(ctx, f.Flow.ID)
	if err != nil {
		return nil, &FlowError{Op: "Finish", FlowName: f.Flow.Name, Err: err}
	}

	var discrepancies []Discrepancy
	errorCount := 0

	maxLen := len(points)
	if len(assertions) > maxLen {
		maxLen = len(assertions)
	}

	for i := 0; i < maxLen; i++ {
		if i >= len(assertions) {
			errorCount++
			discrepancies = append(discrepancies, Discrepancy{
				PointID:     points[i].ID,
				Description: points[i].Description,
				Diff:        "Missing assertion for this point",
				Timestamp:   time.Now(),
			})
			continue
		}

		if i >= len(points) {
			errorCount++
			discrepancies = append(discrepancies, Discrepancy{
				AssertionID: assertions[i].ID,
				Description: "Orphan Assertion",
				Diff:        fmt.Sprintf("Assertion #%d found without a matching Point #%d", i+1, i+1),
				Timestamp:   time.Now(),
			})
			continue
		}

		p := points[i]
		a := assertions[i]

		diffs, equal := DeepCompare(p.Expected, a.Actual)
		if !equal {
			errorCount++
			var expectedVal, actualVal interface{}
			_ = json.Unmarshal(p.Expected, &expectedVal)
			_ = json.Unmarshal(a.Actual, &actualVal)

			diffStr := FormatDiffs(diffs)
			discrepancies = append(discrepancies, Discrepancy{
				PointID:     p.ID,
				AssertionID: a.ID,
				Description: p.Description,
				Expected:    expectedVal,
				Actual:      actualVal,
				Diff:        diffStr,
				Timestamp:   time.Now(),
			})
		}
	}

	executionTime := time.Since(f.startTime)

	result := &FinishResult{
		Success:       len(discrepancies) == 0,
		Discrepancies: discrepancies,
		ExecutionTime: executionTime,
		ErrorCount:    errorCount,
	}

	if result.Success {
		f.client.logger.Info("Flow '%s' finished: SUCCESS (%s)", f.Flow.Name, executionTime)
	} else {
		f.client.logger.Error("Flow '%s' finished: FAILED with %d discrepancies (%s)", f.Flow.Name, errorCount, executionTime)
	}

	return result, nil
}
