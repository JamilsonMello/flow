package main

import (
	"fmt"

	// Import the Flow SDK
	"flow-tool/pkg/flow"

	// Driver for database
	"database/sql"

	_ "github.com/lib/pq"
)

// OrderService wraps your business logic
type OrderService struct {
	// Business dependencies (Repositories, etc.)

	// Flow Client for observability (optional/separate)
	flowClient *flow.FlowClient
}

// CreateOrder is your actual business logic function
func (s *OrderService) CreateOrder(orderID string, amount float64) error {
	// 1. Start Flow Tracking (Non-intrusive, can be wrapped)
	// You might want to do this via a middleware or a decorator
	f, _ := s.flowClient.Start(orderID)

	// 2. Define Expectations (Contract)
	// This is "metadata" about what your business logic *promises* to do
	f.CreatePoint("Order Created", map[string]interface{}{
		"id":     orderID,
		"amount": amount,
		"status": "PENDING",
	})

	// 3. ACTUAL BUSINESS LOGIC
	fmt.Printf("Executing Business Logic for Order %s...\n", orderID)
	// db.Save(order)...
	// queue.Publish(order)...

	return nil
}

func main() {
	// Setup DB
	db, _ := sql.Open("postgres", "user=user password=password dbname=flow_db sslmode=disable host=127.0.0.1 port=5432")

	// Initialize Flow Client
	flowClient, _ := flow.NewClient(db, "OrderService")

	// Inject into Service
	svc := &OrderService{flowClient: flowClient}

	// Run Business Logic
	svc.CreateOrder("ORD-555", 99.90)
}
