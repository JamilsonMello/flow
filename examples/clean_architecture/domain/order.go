package domain

// Order represents the core entity
type Order struct {
	ID     string
	Amount float64
	Status string
}

// OrderRepository interface (Ports)
type OrderRepository interface {
	Save(order Order) error
}

// OrderObserver is an interface for ANYONE who wants to know about order events.
// The domain DOES NOT know about "Flow", "Datadog", "Prometheus", etc.
type OrderObserver interface {
	OnOrderCreated(order Order)
}
