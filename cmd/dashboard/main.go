package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Flow struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Point struct {
	ID          int64           `json:"id"`
	FlowID      int64           `json:"flow_id"`
	Description string          `json:"description"`
	Expected    json.RawMessage `json:"expected"`
	ServiceName string          `json:"service_name"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Assertion struct {
	ID          int64           `json:"id"`
	FlowID      int64           `json:"flow_id"`
	Actual      json.RawMessage `json:"actual"`
	ServiceName string          `json:"service_name"`
	CreatedAt   time.Time       `json:"created_at"`
}

type TimelineEvent struct {
	Type      string      `json:"type"` // "POINT" or "ASSERTION"
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

func main() {
	connStr := "user=user password=password dbname=flow_db sslmode=disable host=127.0.0.1 port=5432"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	fs := http.FileServer(http.Dir("./cmd/dashboard/static"))
	http.Handle("/", fs)

	http.HandleFunc("/api/flows", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w)

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 {
			limit = 20
		}
		offset := (page - 1) * limit

		var total int
		db.QueryRow("SELECT COUNT(*) FROM flows").Scan(&total)

		rows, err := db.Query("SELECT id, name, status, created_at FROM flows ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		var flows []Flow = []Flow{}
		for rows.Next() {
			var f Flow
			if err := rows.Scan(&f.ID, &f.Name, &f.Status, &f.CreatedAt); err != nil {
				continue
			}
			flows = append(flows, f)
		}

		response := map[string]interface{}{
			"data": flows,
			"meta": map[string]interface{}{
				"page":  page,
				"limit": limit,
				"total": total,
				"pages": (total + limit - 1) / limit,
			},
		}
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/api/flows/", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w)
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 {
			http.Error(w, "Invalid ID", 400)
			return
		}
		idStr := parts[3]
		flowID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", 400)
			return
		}

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 {
			limit = 50
		}
		offset := (page - 1) * limit

		var totalPoints, totalAssertions int
		db.QueryRow("SELECT COUNT(*) FROM points WHERE flow_id = $1", flowID).Scan(&totalPoints)
		db.QueryRow("SELECT COUNT(*) FROM assertions WHERE flow_id = $1", flowID).Scan(&totalAssertions)

		var timeline []TimelineEvent = []TimelineEvent{}

		pRows, err := db.Query("SELECT id, description, expected, service_name, created_at FROM points WHERE flow_id = $1 ORDER BY id ASC LIMIT $2 OFFSET $3", flowID, limit, offset)
		if err == nil {
			defer pRows.Close()
			for pRows.Next() {
				var p Point
				var exp []byte
				pRows.Scan(&p.ID, &p.Description, &exp, &p.ServiceName, &p.CreatedAt)
				p.FlowID = flowID
				if exp != nil {
					p.Expected = json.RawMessage(exp)
				}
				timeline = append(timeline, TimelineEvent{Type: "POINT", Timestamp: p.CreatedAt, Data: p})
			}
		}

		aRows, err := db.Query("SELECT id, actual, service_name, created_at FROM assertions WHERE flow_id = $1 ORDER BY id ASC LIMIT $2 OFFSET $3", flowID, limit, offset)
		if err == nil {
			defer aRows.Close()
			for aRows.Next() {
				var a Assertion
				var act []byte
				aRows.Scan(&a.ID, &act, &a.ServiceName, &a.CreatedAt)
				a.FlowID = flowID
				if act != nil {
					a.Actual = json.RawMessage(act)
				}
				timeline = append(timeline, TimelineEvent{Type: "ASSERTION", Timestamp: a.CreatedAt, Data: a})
			}
		}

		sort.SliceStable(timeline, func(i, j int) bool {
			return timeline[i].Timestamp.Before(timeline[j].Timestamp)
		})

		response := map[string]interface{}{
			"data": timeline,
			"meta": map[string]interface{}{
				"page":             page,
				"limit":            limit,
				"total_points":     totalPoints,
				"total_assertions": totalAssertions,
				"total_items":      totalPoints,
				"pages":            (totalPoints + limit - 1) / limit,
			},
		}

		json.NewEncoder(w).Encode(response)
	})

	fmt.Println("Dashboard running at http://localhost:8585")
	log.Fatal(http.ListenAndServe(":8585", nil))
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
}
