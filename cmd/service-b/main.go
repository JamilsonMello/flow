package main

import (
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

	client, err := flow.NewClient(db, "Service B (Logistics & Finance)", false)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// In a real scenario, Service B would receive the ID via queue/http.
	// Here we just pick the latest ACTIVE flow from DB to simulate processing the one just created.
	var flowName string
	err = db.QueryRow("SELECT name FROM flows WHERE status = 'ACTIVE' ORDER BY created_at DESC LIMIT 1").Scan(&flowName)
	if err != nil {
		log.Fatalf("No active flow found to process. Run Service A first!")
	}

	fmt.Printf("Retrieving flow '%s'...\n", flowName)
	f, err := client.GetFlow(flowName)
	if err != nil {
		log.Fatalf("Failed to get flow: %v", err)
	}

	fmt.Println("Processing pipeline...")

	// 1. Acknowledge Receipt
	time.Sleep(500 * time.Millisecond)
	fmt.Println("<- [1] Received")
	f.AddAssertion(map[string]interface{}{
		"status":      "PENDING",
		"total":       150.50,
		"customer_id": "CUST-99",
	})

	// 2. Risk Check Result (Consumer)
	time.Sleep(800 * time.Millisecond)
	fmt.Println("<- [2] Risk Verified")
	f.AddAssertion(map[string]interface{}{
		"risk_score": 0.05,
		"approved":   true,
		"source":     "internal-ai",
	})

	// 3. Finance Processing
	time.Sleep(600 * time.Millisecond)
	fmt.Println("<- [3] Payment Confirmed")
	f.AddAssertion(map[string]interface{}{
		"provider": "Stripe",
		"status":   "CAPTURED",
		"amount":   150.50,
	})

	// 4. Warehouse Processing
	time.Sleep(700 * time.Millisecond)
	fmt.Println("<- [4] Warehouse Acceptance")
	f.AddAssertion(map[string]interface{}{
		"warehouse":          "SP-01",
		"priority":           "HIGH",
		"estimated_delivery": "2026-02-10",
	})

	fmt.Println("Finishing flow validation...")
	result, err := f.Finish()
	if err != nil {
		log.Fatalf("Failed to finish flow: %v", err)
	}

	if result.Success {
		fmt.Println("✅ ----------------------------------")
		fmt.Println("✅ Flow validation PASSED PERFECTLY")
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
