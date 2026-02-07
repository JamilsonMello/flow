package main

import (
	"context"
	"fmt"

	"flow-tool/pkg/flow"
)

type Message struct {
	ID      string
	Payload map[string]interface{}
}

type Handler func(ctx context.Context, msg Message) error

func FlowMiddleware(flowClient *flow.FlowClient, next Handler) Handler {
	return func(ctx context.Context, msg Message) error {
		flowID := msg.ID

		fmt.Printf("[Middleware] Creating assertion for Flow: %s\n", flowID)

		f, err := flowClient.GetFlow(flowID)
		if err == nil {
			f.AddAssertion(msg.Payload)
		} else {
			fmt.Printf("[Middleware] Warning: Flow %s not found\n", flowID)
		}

		return next(ctx, msg)
	}
}

func main() {
	// db, _ := sql.Open(...)
	// flowClient, _ := flow.NewClient(db, "ConsumerService", false)

	var flowClient *flow.FlowClient = nil

	myBusinessHandler := func(ctx context.Context, msg Message) error {
		fmt.Printf(">> Processing Business Logic for: %v\n", msg.Payload)
		return nil
	}

	decoratedHandler := FlowMiddleware(flowClient, myBusinessHandler)

	msg := Message{
		ID: "ORD-123",
		Payload: map[string]interface{}{
			"amount": 100,
			"status": "paid",
		},
	}

	if flowClient != nil {
		decoratedHandler(context.Background(), msg)
	} else {
		fmt.Println("Mock setup complete. Connect DB to run.")
	}
}
