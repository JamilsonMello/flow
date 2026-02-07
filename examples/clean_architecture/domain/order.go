package domain

// Order represents the core entity
type Order struct {
	ID     string
	Amount float64
	Status string
}

type OrderRepository interface {
	Save(order Order) error
}

type OrderObserver interface {
	OnOrderCreated(order Order)
}
