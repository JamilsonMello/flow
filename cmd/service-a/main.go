package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"flow-tool/pkg/flow"

	_ "github.com/lib/pq"
)

func main() {
	connStr := "user=user password=password dbname=flow_db sslmode=disable host=127.0.0.1 port=5432"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	time.Sleep(2 * time.Second)

	client, err := flow.NewClient(db, flow.FlowConfig{
		ServiceName:   "Service A (Order System)",
		IsProduction:  false,
		MaxExecutions: 2,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	rand.Seed(time.Now().UnixNano())
	orderID := fmt.Sprintf("ORDER-%d", rand.Intn(99999))
	if len(os.Args) > 1 {
		orderID = os.Args[1]
	}

	ctx := context.Background()

	fmt.Printf("Starting flow '%s'...\n", orderID)

	f, err := client.Start(ctx, "Order Processing", orderID)
	if err != nil {
		log.Fatalf("Failed to start flow: %v", err)
	}

	// 1. Order Received
	fmt.Println("-> [1] Order Received")
	err = f.CreatePoint(ctx, "Step 1: Order Received", map[string]interface{}{
		"status":      "PENDING",
		"total":       150.50,
		"customer_id": "CUST-99",
	})
	checkErr(err)
	time.Sleep(500 * time.Millisecond)

	// 2. Risk Check
	fmt.Println("-> [2] Risk Check Passed")
	err = f.CreatePoint(ctx, "Step 2: Risk Analysis", map[string]interface{}{
		"risk_score": 0.05,
		"approved":   true,
		"source":     "internal-ai",
	})
	checkErr(err)
	time.Sleep(500 * time.Millisecond)

	// 3. Payment Authorization
	fmt.Println("-> [3] Payment Authorized")
	err = f.CreatePoint(ctx, "Step 3: Payment", map[string]interface{}{
		"provider": "Stripe",
		"status":   "CAPTURED",
		"amount":   150.50,
	})
	checkErr(err)
	time.Sleep(500 * time.Millisecond)

	// 4. Handover to Logistics
	fmt.Println("-> [4] Sent to Logistics")
	err = f.CreatePoint(ctx, "Step 4: Logistics Handover", map[string]interface{}{
		"warehouse":          "SP-01",
		"priority":           "HIGH",
		"estimated_delivery": "2026-02-10",
	})
	checkErr(err)

	fmt.Printf("Service A completed. Flow '%s' is ready for Service B.\n", orderID)
}

func checkErr(err error) {
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
