package flow

import (
	"context"
	"database/sql"
	"time"
)

type FlowTracker interface {
	Start(ctx context.Context, flowName string, identifier ...string) (FlowExecutor, error)
	GetFlow(ctx context.Context, flowName string, identifier ...string) (FlowExecutor, error)
	Close() error
}

type FlowExecutor interface {
	CreatePoint(ctx context.Context, description string, expected interface{}, opts ...PointOption) error
	AddAssertion(ctx context.Context, actual interface{}) error
	Finish(ctx context.Context) (*FinishResult, error)
	GetFlowInfo() *Flow
}

type Validator interface {
	Validate(expected, actual interface{}) (string, bool)
}

type Storage interface {
	SaveFlow(ctx context.Context, flow *Flow) error
	UpdateFlowStatus(ctx context.Context, flowID int64, status string) error
	SavePoint(ctx context.Context, point *Point) error
	SaveAssertion(ctx context.Context, assertion *Assertion) error
	GetFlow(ctx context.Context, flowName, identifier string) (*Flow, error)
	GetPoints(ctx context.Context, flowID int64) ([]Point, error)
	GetAssertions(ctx context.Context, flowID int64) ([]Assertion, error)
	Close() error
}

type FlowConfig struct {
	ServiceName   string
	IsProduction  bool
	MaxExecutions int
	StorageConfig StorageConfig
	SchemaEnabled bool
	BatchSize     int
	CacheEnabled  bool
	MaxCacheSize  int
	Timeout       time.Duration
}

type StorageConfig struct {
	DB             *sql.DB
	ConnectionPool PoolConfig
	TableName      string
}

type PoolConfig struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}
