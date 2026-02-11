package flow

import (
	"testing"
	"time"
)

func TestIsSkipped(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"SKIPPED", true},
		{"SKIPPED_LIMIT", true},
		{"ACTIVE", false},
		{"FINISHED", false},
		{"", false},
		{"SKIP", false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := isSkipped(tt.status)
			if got != tt.want {
				t.Errorf("isSkipped(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestFlowCache(t *testing.T) {
	t.Run("Disabled cache returns no hits", func(t *testing.T) {
		c := newFlowCache(false, 10)
		c.Set("test", "id", &Flow{Name: "test"})

		_, ok := c.Get("test", "id")
		if ok {
			t.Error("disabled cache should never return a hit")
		}
	})

	t.Run("Enabled cache stores and retrieves", func(t *testing.T) {
		c := newFlowCache(true, 10)
		f := &Flow{ID: 1, Name: "test"}
		c.Set("test", "id", f)

		got, ok := c.Get("test", "id")
		if !ok {
			t.Fatal("expected cache hit")
		}
		if got.ID != 1 {
			t.Errorf("got ID %d, want 1", got.ID)
		}
	})

	t.Run("Delete removes entry", func(t *testing.T) {
		c := newFlowCache(true, 10)
		c.Set("test", "id", &Flow{Name: "test"})
		c.Delete("test", "id")

		_, ok := c.Get("test", "id")
		if ok {
			t.Error("deleted entry should not be found")
		}
	})

	t.Run("Eviction when full", func(t *testing.T) {
		c := newFlowCache(true, 2)
		c.Set("a", "", &Flow{Name: "a"})
		c.Set("b", "", &Flow{Name: "b"})
		c.Set("c", "", &Flow{Name: "c"}) // should evict one

		count := 0
		for _, name := range []string{"a", "b", "c"} {
			if _, ok := c.Get(name, ""); ok {
				count++
			}
		}
		if count != 2 {
			t.Errorf("expected 2 entries after eviction, got %d", count)
		}
	})

	t.Run("Clear empties cache", func(t *testing.T) {
		c := newFlowCache(true, 10)
		c.Set("test", "id", &Flow{Name: "test"})
		c.Clear()

		_, ok := c.Get("test", "id")
		if ok {
			t.Error("cleared cache should not return hits")
		}
	})
}

func TestCacheKey(t *testing.T) {
	got := cacheKey("order-flow", "ORD-123")
	want := "order-flow::ORD-123"
	if got != want {
		t.Errorf("cacheKey() = %q, want %q", got, want)
	}
}

func TestFlowErrorFormat(t *testing.T) {
	err := &FlowError{
		Op:       "Start",
		FlowName: "order-flow",
		Err:      ErrFlowNotFound,
	}

	want := "flow.Start [order-flow]: flow: not found"
	if err.Error() != want {
		t.Errorf("FlowError.Error() = %q, want %q", err.Error(), want)
	}

	if !IsNotFound(err) {
		t.Error("IsNotFound should return true for ErrFlowNotFound")
	}
}

func TestFlowErrorUnwrap(t *testing.T) {
	err := &FlowError{
		Op:  "GetFlow",
		Err: ErrLimitReached,
	}

	if !IsLimitReached(err) {
		t.Error("IsLimitReached should return true")
	}
	if IsNotFound(err) {
		t.Error("IsNotFound should return false for ErrLimitReached")
	}
}

func TestClientBuilder(t *testing.T) {
	b := NewClientBuilder().
		WithServiceName("test-service").
		WithProductionMode(false).
		WithMaxExecutions(5).
		WithSchemaValidation(true).
		WithBatchSize(50).
		WithCaching(true, 500).
		WithTimeout(10*time.Second).
		WithConnectionPool(5, 25, 5*time.Minute).
		WithLogger(NewStdLogger())

	if b.config.ServiceName != "test-service" {
		t.Errorf("ServiceName = %s, want test-service", b.config.ServiceName)
	}
	if b.config.MaxExecutions != 5 {
		t.Errorf("MaxExecutions = %d, want 5", b.config.MaxExecutions)
	}
	if !b.config.SchemaEnabled {
		t.Error("SchemaEnabled should be true")
	}
	if !b.config.CacheEnabled {
		t.Error("CacheEnabled should be true")
	}
	if b.config.MaxCacheSize != 500 {
		t.Errorf("MaxCacheSize = %d, want 500", b.config.MaxCacheSize)
	}
	if b.config.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", b.config.Timeout)
	}

	pool := b.config.StorageConfig.ConnectionPool
	if pool.MaxIdleConns != 5 {
		t.Errorf("MaxIdleConns = %d, want 5", pool.MaxIdleConns)
	}
	if pool.MaxOpenConns != 25 {
		t.Errorf("MaxOpenConns = %d, want 25", pool.MaxOpenConns)
	}
	if b.logger == nil {
		t.Error("logger should not be nil")
	}
}

func TestPointOptions(t *testing.T) {
	schema := []byte(`{"type":"object"}`)
	timeout := 5 * time.Second

	p := &Point{}
	WithSchema(schema)(p)
	WithTimeout(timeout)(p)

	if string(p.Schema) != string(schema) {
		t.Error("WithSchema failed")
	}
	if p.Timeout == nil || *p.Timeout != timeout {
		t.Error("WithTimeout failed")
	}
}

func TestClientBuilderNoDB(t *testing.T) {
	_, err := NewClientBuilder().Build()
	if err == nil {
		t.Error("Build without DB should return error")
	}
}
