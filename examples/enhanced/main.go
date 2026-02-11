package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	"flow-tool/pkg/flow"

	_ "github.com/lib/pq"
)

func main() {
	// Connect to DB
	connStr := "user=user password=password dbname=flow_db sslmode=disable host=127.0.0.1 port=5432"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	// Wait for DB to be ready given docker-compose startup
	time.Sleep(2 * time.Second)

	// Use the enhanced builder pattern to create a client
	client, err := flow.NewClientBuilder().
		WithDB(db).
		WithServiceName("Enhanced Order Service").
		WithProductionMode(false).
		WithMaxExecutions(100).
		WithSchemaValidation(true).
		WithCaching(true, 500).
		WithConnectionPool(5, 20, 30*time.Minute).
		Build()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Generate a random Order ID
	rand.Seed(time.Now().UnixNano())
	orderID := fmt.Sprintf("ENHANCED-ORDER-%d", rand.Intn(99999))

	fmt.Printf("Starting enhanced flow '%s'...\n", orderID)

	ctx := context.Background()
	
	// Start the flow
	f, err := client.Start(ctx, "Enhanced Order Processing", orderID)
	if err != nil {
		log.Fatalf("Failed to start flow: %v", err)
	}

	// Create points with enhanced options
	fmt.Println("-> [1] Order Received")
	err = f.CreatePoint(ctx, "Step 1: Order Received", map[string]interface{}{
		"status":      "PENDING",
		"total":       150.50,
		"customer_id": "CUST-99",
	}, flow.WithTimeout(5*time.Second))
	checkErr(err)

	// Create point with schema validation
	orderSchema := []byte(`{
		"type": "object",
		"properties": {
			"status": {"type": "string"},
			"total": {"type": "number"},
			"customer_id": {"type": "string"}
		},
		"required": ["status", "total", "customer_id"]
	}`)
	
	fmt.Println("-> [2] Risk Check Passed")
	err = f.CreatePoint(ctx, "Step 2: Risk Analysis", map[string]interface{}{
		"risk_score": 0.05,
		"approved":   true,
		"source":     "internal-ai",
	}, flow.WithSchema(orderSchema))
	checkErr(err)

	// Add more points
	fmt.Println("-> [3] Payment Authorized")
	err = f.CreatePoint(ctx, "Step 3: Payment", map[string]interface{}{
		"provider": "Stripe",
		"status":   "CAPTURED",
		"amount":   150.50,
	})
	checkErr(err)

	fmt.Println("-> [4] Sent to Logistics")
	err = f.CreatePoint(ctx, "Step 4: Logistics Handover", map[string]interface{}{
		"warehouse":          "SP-01",
		"priority":           "HIGH",
		"estimated_delivery": "2026-02-10",
	})
	checkErr(err)

	fmt.Printf("Enhanced Service A completed. Flow '%s' is ready for Service B.\n", orderID)

	// Simulate Service B processing
	fmt.Println("\n=== Simulating Service B Processing ===")
	
	// Retrieve the same flow in Service B
	serviceBClient, err := flow.NewClientBuilder().
		WithDB(db).
		WithServiceName("Enhanced Logistics Service").
		WithProductionMode(false).
		WithMaxExecutions(100).
		Build()
	if err != nil {
		log.Fatalf("Failed to create Service B client: %v", err)
	}
	defer serviceBClient.Close()

	serviceBFlow, err := serviceBClient.GetFlow(ctx, "Enhanced Order Processing", orderID)
	if err != nil {
		log.Fatalf("Failed to get flow in Service B: %v", err)
	}

	fmt.Println("Processing pipeline in Service B...")

	// Add assertions
	fmt.Println("<- [1] Received in Service B")
	serviceBFlow.AddAssertion(ctx, map[string]interface{}{
		"status":      "PENDING",
		"total":       150.50,
		"customer_id": "CUST-99",
	})

	fmt.Println("<- [2] Risk Verified in Service B")
	serviceBFlow.AddAssertion(ctx, map[string]interface{}{
		"risk_score": 0.05,
		"approved":   true,
		"source":     "internal-ai",
	})

	fmt.Println("<- [3] Payment Confirmed in Service B")
	serviceBFlow.AddAssertion(ctx, map[string]interface{}{
		"provider": "Stripe",
		"status":   "CAPTURED",
		"amount":   150.50,
	})

	fmt.Println("<- [4] Warehouse Acceptance in Service B")
	serviceBFlow.AddAssertion(ctx, map[string]interface{}{
		"warehouse":          "SP-01",
		"priority":           "HIGH",
		"estimated_delivery": "2026-02-10",
	})

	fmt.Println("Finishing enhanced flow validation...")
	result, err := serviceBFlow.Finish(ctx)
	if err != nil {
		log.Fatalf("Failed to finish flow: %v", err)
	}

	if result.Success {
		fmt.Println("✅ ----------------------------------")
		fmt.Println("✅ Enhanced Flow validation PASSED")
		fmt.Printf("✅ Execution Time: %v\n", result.ExecutionTime)
		fmt.Printf("✅ Error Count: %d\n", result.ErrorCount)
		fmt.Println("✅ ----------------------------------")
	} else {
		fmt.Println("❌ Enhanced Flow validation FAILED!")
		for _, d := range result.Discrepancies {
			fmt.Printf("\n[DISCREPANCY] Point: %s\n", d.Description)
			if d.Diff != "" {
				fmt.Printf("  Diff: %s\n", d.Diff)
			} else {
				fmt.Printf("  Error: %s\n", d.Description)
			}
		}
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
