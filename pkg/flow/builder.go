package flow

import (
	"database/sql"
	"time"
)

type ClientBuilder struct {
	config FlowConfig
	db     *sql.DB
	logger Logger
}

func NewClientBuilder() *ClientBuilder {
	return &ClientBuilder{
		config: FlowConfig{
			MaxExecutions: 0,
			SchemaEnabled: false,
			BatchSize:     100,
			CacheEnabled:  false,
			MaxCacheSize:  1000,
			Timeout:       30 * time.Second,
		},
	}
}

func (b *ClientBuilder) WithDB(db *sql.DB) *ClientBuilder {
	b.db = db
	return b
}

func (b *ClientBuilder) WithServiceName(serviceName string) *ClientBuilder {
	b.config.ServiceName = serviceName
	return b
}

func (b *ClientBuilder) WithProductionMode(isProduction bool) *ClientBuilder {
	b.config.IsProduction = isProduction
	return b
}

func (b *ClientBuilder) WithMaxExecutions(max int) *ClientBuilder {
	b.config.MaxExecutions = max
	return b
}

func (b *ClientBuilder) WithSchemaValidation(enabled bool) *ClientBuilder {
	b.config.SchemaEnabled = enabled
	return b
}

func (b *ClientBuilder) WithBatchSize(size int) *ClientBuilder {
	b.config.BatchSize = size
	return b
}

func (b *ClientBuilder) WithCaching(enabled bool, maxSize int) *ClientBuilder {
	b.config.CacheEnabled = enabled
	b.config.MaxCacheSize = maxSize
	return b
}

func (b *ClientBuilder) WithTimeout(timeout time.Duration) *ClientBuilder {
	b.config.Timeout = timeout
	return b
}

func (b *ClientBuilder) WithConnectionPool(maxIdle, maxOpen int, maxLifetime time.Duration) *ClientBuilder {
	b.config.StorageConfig.ConnectionPool = PoolConfig{
		MaxIdleConns:    maxIdle,
		MaxOpenConns:    maxOpen,
		ConnMaxLifetime: maxLifetime,
	}
	return b
}

func (b *ClientBuilder) WithLogger(logger Logger) *ClientBuilder {
	b.logger = logger
	return b
}

func (b *ClientBuilder) Build() (*FlowClient, error) {
	if b.db == nil {
		return nil, &ConfigError{msg: "database connection is required"}
	}

	pool := b.config.StorageConfig.ConnectionPool
	if pool.MaxIdleConns > 0 {
		b.db.SetMaxIdleConns(pool.MaxIdleConns)
	}
	if pool.MaxOpenConns > 0 {
		b.db.SetMaxOpenConns(pool.MaxOpenConns)
	}
	if pool.ConnMaxLifetime > 0 {
		b.db.SetConnMaxLifetime(pool.ConnMaxLifetime)
	}

	b.config.StorageConfig.DB = b.db
	client, err := NewClient(b.db, b.config)
	if err != nil {
		return nil, err
	}
	if b.logger != nil {
		client.logger = b.logger
	}
	return client, nil
}

type ConfigError struct {
	msg string
}

func (e *ConfigError) Error() string {
	return e.msg
}
