package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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

	client, err := flow.NewClient(db, flow.FlowConfig{
		ServiceName:   "Service B (Logistics & Finance)",
		IsProduction:  false,
		MaxExecutions: 2,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Discover the most recent active flow (simulates receiving the order ID from a queue/API)
	var flowName string
	var identifier sql.NullString
	err = db.QueryRow("SELECT name, identifier FROM flows WHERE status = 'ACTIVE' ORDER BY created_at DESC LIMIT 1").Scan(&flowName, &identifier)
	if err != nil {
		// No active flow — check if the flow hit its execution limit (zero overhead path)
		flowName = "Order Processing"
		ctx := context.Background()
		f, getErr := client.GetFlow(ctx, flowName)
		if getErr != nil {
			log.Fatalf("No active flow found to process. Run Service A first!")
		}
		// GetFlow returned a SKIPPED_LIMIT instance — nothing to do
		info := f.GetFlowInfo()
		fmt.Printf("⚠ Flow '%s' reached execution limit (status: %s). Nothing to process.\n", flowName, info.Status)
		return
	}

	ctx := context.Background()

	fmt.Printf("Retrieving flow '%s' (ID: %s)...\n", flowName, identifier.String)
	f, err := client.GetFlow(ctx, flowName, identifier.String)
	if err != nil {
		log.Fatalf("Failed to get flow: %v", err)
	}

	// Check if flow was skipped due to limit
	info := f.GetFlowInfo()
	if info.Status == "SKIPPED_LIMIT" || info.Status == "SKIPPED" {
		fmt.Printf("⚠ Flow '%s' is skipped (status: %s). Nothing to process.\n", flowName, info.Status)
		return
	}

	fmt.Println("Processing pipeline...")

	// 1. Acknowledge Receipt
	time.Sleep(500 * time.Millisecond)
	fmt.Println("<- [1] Received")
	f.AddAssertion(ctx, map[string]interface{}{
		"status":      "PENDING",
		"total":       150.50,
		"customer_id": "CUST-99",
	})

	// 2. Risk Check Result
	time.Sleep(800 * time.Millisecond)
	fmt.Println("<- [2] Risk Verified")
	f.AddAssertion(ctx, map[string]interface{}{
		"risk_score": 0.05,
		"approved":   true,
		"source":     "internal-ai",
	})

	// 3. Finance Processing
	time.Sleep(600 * time.Millisecond)
	fmt.Println("<- [3] Payment Confirmed")
	f.AddAssertion(ctx, map[string]interface{}{
		"provider": "Stripe",
		"status":   "CAPTURED",
		"amount":   150.50,
	})

	// 4. Warehouse Processing
	time.Sleep(700 * time.Millisecond)
	fmt.Println("<- [4] Warehouse Acceptance")
	f.AddAssertion(ctx, map[string]interface{}{
		"warehouse":          "SP-01",
		"priority":           "HIGH",
		"estimated_delivery": "2026-02-10",
	})

	fmt.Println("Finishing flow validation...")
	result, err := f.Finish(ctx)
	if err != nil {
		log.Fatalf("Failed to finish flow: %v", err)
	}

	if result.Success {
		fmt.Println("✅ ----------------------------------")
		fmt.Println("✅ Flow validation PASSED PERFECTLY")
		fmt.Printf("✅ Execution Time: %v\n", result.ExecutionTime)
		fmt.Println("✅ ----------------------------------")
	} else {
		fmt.Println("❌ Flow validation FAILED!")
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
