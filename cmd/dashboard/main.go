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

	"flow-tool/pkg/config"
	"flow-tool/pkg/flow"

	_ "github.com/lib/pq"
)

type TimelineEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type Stats struct {
	TotalFlows       int `json:"total_flows"`
	ActiveFlows      int `json:"active_flows"`
	FinishedFlows    int `json:"finished_flows"`
	InterruptedFlows int `json:"interrupted_flows"`
	TotalPoints      int `json:"total_points"`
	TotalAssertions  int `json:"total_assertions"`
}

func main() {
	cfg, err := config.LoadConfig("flow.config.yaml")
	if err != nil {
		cfg, err = config.LoadConfig("../../flow.config.yaml")
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	}

	db, err := sql.Open("postgres", cfg.GetConnString())
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	fs := http.FileServer(http.Dir("./cmd/dashboard/static"))
	http.Handle("/", fs)

	// ───── GET /api/stats ─────
	http.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w)
		var s Stats
		db.QueryRow("SELECT COUNT(*) FROM flows").Scan(&s.TotalFlows)
		db.QueryRow("SELECT COUNT(*) FROM flows WHERE status='ACTIVE'").Scan(&s.ActiveFlows)
		db.QueryRow("SELECT COUNT(*) FROM flows WHERE status='FINISHED'").Scan(&s.FinishedFlows)
		db.QueryRow("SELECT COUNT(*) FROM flows WHERE status='INTERRUPTED'").Scan(&s.InterruptedFlows)
		db.QueryRow("SELECT COUNT(*) FROM points").Scan(&s.TotalPoints)
		db.QueryRow("SELECT COUNT(*) FROM assertions").Scan(&s.TotalAssertions)
		json.NewEncoder(w).Encode(s)
	})

	// ───── GET /api/flows ─────
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

		statusFilter := r.URL.Query().Get("status")
		search := r.URL.Query().Get("search")

		where := "WHERE 1=1"
		args := []interface{}{}
		argIdx := 1

		if statusFilter != "" {
			where += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, statusFilter)
			argIdx++
		}
		if search != "" {
			where += fmt.Sprintf(" AND (name ILIKE $%d OR identifier ILIKE $%d OR service ILIKE $%d)", argIdx, argIdx, argIdx)
			args = append(args, "%"+search+"%")
			argIdx++
		}

		var total int
		countArgs := make([]interface{}, len(args))
		copy(countArgs, args)
		db.QueryRow("SELECT COUNT(*) FROM flows "+where, countArgs...).Scan(&total)

		query := fmt.Sprintf(
			"SELECT id, name, identifier, status, service, created_at, updated_at FROM flows %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
			where, argIdx, argIdx+1,
		)
		args = append(args, limit, offset)

		rows, err := db.Query(query, args...)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		var flows []flow.Flow = []flow.Flow{}
		for rows.Next() {
			var f flow.Flow
			var ident, svc sql.NullString
			var updatedAt sql.NullTime
			if err := rows.Scan(&f.ID, &f.Name, &ident, &f.Status, &svc, &f.CreatedAt, &updatedAt); err != nil {
				continue
			}
			f.Identifier = ident.String
			f.Service = svc.String
			if updatedAt.Valid {
				f.UpdatedAt = updatedAt.Time
			}
			flows = append(flows, f)
		}

		// Count points and assertions per flow
		type FlowWithCounts struct {
			flow.Flow
			PointCount     int `json:"point_count"`
			AssertionCount int `json:"assertion_count"`
		}

		enriched := make([]FlowWithCounts, len(flows))
		for i, f := range flows {
			enriched[i].Flow = f
			db.QueryRow("SELECT COUNT(*) FROM points WHERE flow_id = $1", f.ID).Scan(&enriched[i].PointCount)
			db.QueryRow("SELECT COUNT(*) FROM assertions WHERE flow_id = $1", f.ID).Scan(&enriched[i].AssertionCount)
		}

		response := map[string]interface{}{
			"data": enriched,
			"meta": map[string]interface{}{
				"page":  page,
				"limit": limit,
				"total": total,
				"pages": (total + limit - 1) / limit,
			},
		}
		json.NewEncoder(w).Encode(response)
	})

	// ───── GET /api/flows/:id ─────
	http.HandleFunc("/api/flows/", func(w http.ResponseWriter, r *http.Request) {
		enableCors(w)
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 {
			http.Error(w, "Invalid ID", 400)
			return
		}
		idStr := parts[3]

		// /api/flows/:id/compare
		if len(parts) >= 5 && parts[4] == "compare" {
			handleCompare(db, w, idStr)
			return
		}

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

		// Flow info
		var flowInfo flow.Flow
		var ident, svc sql.NullString
		var updatedAt sql.NullTime
		db.QueryRow("SELECT id, name, identifier, status, service, created_at, updated_at FROM flows WHERE id = $1", flowID).
			Scan(&flowInfo.ID, &flowInfo.Name, &ident, &flowInfo.Status, &svc, &flowInfo.CreatedAt, &updatedAt)
		flowInfo.Identifier = ident.String
		flowInfo.Service = svc.String
		if updatedAt.Valid {
			flowInfo.UpdatedAt = updatedAt.Time
		}

		var timeline []TimelineEvent = []TimelineEvent{}

		pRows, err := db.Query(
			"SELECT id, description, expected, service_name, schema, timeout, created_at FROM points WHERE flow_id = $1 ORDER BY id ASC LIMIT $2 OFFSET $3",
			flowID, limit, offset,
		)
		if err == nil {
			defer pRows.Close()
			for pRows.Next() {
				var p flow.Point
				var exp, schema []byte
				var timeoutMs sql.NullInt64
				pRows.Scan(&p.ID, &p.Description, &exp, &p.ServiceName, &schema, &timeoutMs, &p.CreatedAt)
				p.FlowID = flowID
				if exp != nil {
					p.Expected = json.RawMessage(exp)
				}
				if schema != nil {
					p.Schema = json.RawMessage(schema)
				}
				if timeoutMs.Valid {
					d := time.Duration(timeoutMs.Int64) * time.Millisecond
					p.Timeout = &d
				}
				timeline = append(timeline, TimelineEvent{Type: "POINT", Timestamp: p.CreatedAt, Data: p})
			}
		}

		aRows, err := db.Query(
			"SELECT id, actual, service_name, processed_at, created_at FROM assertions WHERE flow_id = $1 ORDER BY id ASC LIMIT $2 OFFSET $3",
			flowID, limit, offset,
		)
		if err == nil {
			defer aRows.Close()
			for aRows.Next() {
				var a flow.Assertion
				var act []byte
				var processedAt sql.NullTime
				aRows.Scan(&a.ID, &act, &a.ServiceName, &processedAt, &a.CreatedAt)
				a.FlowID = flowID
				if act != nil {
					a.Actual = json.RawMessage(act)
				}
				if processedAt.Valid {
					a.ProcessedAt = &processedAt.Time
				}
				timeline = append(timeline, TimelineEvent{Type: "ASSERTION", Timestamp: a.CreatedAt, Data: a})
			}
		}

		sort.SliceStable(timeline, func(i, j int) bool {
			return timeline[i].Timestamp.Before(timeline[j].Timestamp)
		})

		response := map[string]interface{}{
			"data": timeline,
			"flow": flowInfo,
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

	fmt.Printf("Dashboard running at http://localhost:%d\n", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), nil))
}

// handleCompare runs multi-diff comparison for a flow and returns results.
func handleCompare(db *sql.DB, w http.ResponseWriter, idStr string) {
	flowID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", 400)
		return
	}

	// Fetch all points
	pRows, err := db.Query("SELECT id, description, expected FROM points WHERE flow_id = $1 ORDER BY id ASC", flowID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer pRows.Close()

	var points []struct {
		ID          int64
		Description string
		Expected    json.RawMessage
	}
	for pRows.Next() {
		var p struct {
			ID          int64
			Description string
			Expected    json.RawMessage
		}
		var exp []byte
		pRows.Scan(&p.ID, &p.Description, &exp)
		if exp != nil {
			p.Expected = json.RawMessage(exp)
		}
		points = append(points, p)
	}

	// Fetch all assertions
	aRows, err := db.Query("SELECT id, actual FROM assertions WHERE flow_id = $1 ORDER BY id ASC", flowID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer aRows.Close()

	var assertions []struct {
		ID     int64
		Actual json.RawMessage
	}
	for aRows.Next() {
		var a struct {
			ID     int64
			Actual json.RawMessage
		}
		var act []byte
		aRows.Scan(&a.ID, &act)
		if act != nil {
			a.Actual = json.RawMessage(act)
		}
		assertions = append(assertions, a)
	}

	type CompareResult struct {
		Index       int              `json:"index"`
		PointID     int64            `json:"point_id"`
		AssertionID int64            `json:"assertion_id,omitempty"`
		Description string           `json:"description"`
		Match       bool             `json:"match"`
		Diffs       []flow.DiffEntry `json:"diffs,omitempty"`
		Expected    json.RawMessage  `json:"expected,omitempty"`
		Actual      json.RawMessage  `json:"actual,omitempty"`
		Status      string           `json:"status"`
	}

	var results []CompareResult

	maxLen := len(points)
	if len(assertions) > maxLen {
		maxLen = len(assertions)
	}

	for i := 0; i < maxLen; i++ {
		r := CompareResult{Index: i}

		if i >= len(assertions) {
			r.PointID = points[i].ID
			r.Description = points[i].Description
			r.Expected = points[i].Expected
			r.Status = "missing_assertion"
			r.Match = false
		} else if i >= len(points) {
			r.AssertionID = assertions[i].ID
			r.Description = "Orphan Assertion"
			r.Actual = assertions[i].Actual
			r.Status = "orphan_assertion"
			r.Match = false
		} else {
			r.PointID = points[i].ID
			r.AssertionID = assertions[i].ID
			r.Description = points[i].Description
			r.Expected = points[i].Expected
			r.Actual = assertions[i].Actual
			diffs, equal := flow.DeepCompare(points[i].Expected, assertions[i].Actual)
			r.Match = equal
			r.Diffs = diffs
			if equal {
				r.Status = "match"
			} else {
				r.Status = "mismatch"
			}
		}
		results = append(results, r)
	}

	matchCount := 0
	for _, r := range results {
		if r.Match {
			matchCount++
		}
	}

	response := map[string]interface{}{
		"results":       results,
		"total":         len(results),
		"matches":       matchCount,
		"mismatches":    len(results) - matchCount,
		"success":       matchCount == len(results),
		"total_points":  len(points),
		"total_asserts": len(assertions),
	}
	json.NewEncoder(w).Encode(response)
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
}
