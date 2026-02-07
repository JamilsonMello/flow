package main

import (
	"context"
	"fmt"

	// Import the Flow SDK
	"flow-tool/pkg/flow"
)

// Message is a generic representation of a queue message
type Message struct {
	ID      string
	Payload map[string]interface{}
}

// Handler is the function signature for your business logic
type Handler func(ctx context.Context, msg Message) error

// FlowMiddleware intercepts message processing to add Flow assertions
func FlowMiddleware(flowClient *flow.FlowClient, next Handler) Handler {
	return func(ctx context.Context, msg Message) error {
		// 1. Identify context (Correlation ID)
		// Assuming msg.ID is the correlation ID for the flow (e.g. Order ID)
		flowID := msg.ID

		fmt.Printf("[Middleware] Creating assertion for Flow: %s\n", flowID)

		// 2. Retrieve (or implicity reference) the Flow
		f, err := flowClient.GetFlow(flowID)
		if err == nil {
			// 3. Create Assertion based on the incoming message payload
			// This validates that what arrived in this service matches what was expected
			// The description typically identifies the "event" being processed
			f.AddAssertion(msg.Payload)
		} else {
			// If Flow doesn't exist, maybe it wasn't tracked by the sender?
			// You can decide to log, ignore, or create a 'partial' flow.
			fmt.Printf("[Middleware] Warning: Flow %s not found\n", flowID)
		}

		// 4. Pass control to the actual business logic
		return next(ctx, msg)
	}
}

// Example usage
func main() {
	// --- Infrastructure Setup ---
	// db, _ := sql.Open(...)
	// flowClient, _ := flow.NewClient(db, "ConsumerService", false)

	// Mocking for example
	// flowClient := &flow.Client{}
	// To make this runnable without DB, we'd need a mock, but for compilation:
	var flowClient *flow.FlowClient = nil

	// --- Business Logic ---
	myBusinessHandler := func(ctx context.Context, msg Message) error {
		// This code KNOWS NOTHING about Flow
		fmt.Printf(">> Processing Business Logic for: %v\n", msg.Payload)
		return nil
	}

	// --- Wiring ---
	// Wrap the handler with Flow Middleware
	decoratedHandler := FlowMiddleware(flowClient, myBusinessHandler)

	// --- Simulation ---
	msg := Message{
		ID: "ORD-123",
		Payload: map[string]interface{}{
			"amount": 100,
			"status": "paid",
		},
	}

	// Execute (Panic expected if flowClient is nil, but code compiles)
	if flowClient != nil {
		decoratedHandler(context.Background(), msg)
	} else {
		fmt.Println("Mock setup complete. Connect DB to run.")
	}
}
