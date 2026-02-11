package main

import (
	"context"
	"fmt"

	"flow-tool/pkg/flow"

	"database/sql"

	_ "github.com/lib/pq"
)

type OrderService struct {
	flowClient *flow.FlowClient
}

func (s *OrderService) CreateOrder(orderID string, amount float64) error {
	ctx := context.Background()

	f, _ := s.flowClient.Start(ctx, "Create Order Flow", orderID)
	f.CreatePoint(ctx, "Order Created", map[string]interface{}{
		"id":     orderID,
		"amount": amount,
		"status": "PENDING",
	})

	fmt.Printf("Executing Business Logic for Order %s...\n", orderID)

	return nil
}

func main() {
	db, _ := sql.Open("postgres", "user=user password=password dbname=flow_db sslmode=disable host=127.0.0.1 port=5432")

	flowClient, _ := flow.NewClient(db, flow.FlowConfig{
		ServiceName:   "OrderService",
		IsProduction:  false,
		MaxExecutions: 1,
	})

	svc := &OrderService{flowClient: flowClient}

	svc.CreateOrder("ORD-555", 99.90)
}
