package flow

import (
	"encoding/json"
	"testing"
)

func TestDeepCompare(t *testing.T) {
	tests := []struct {
		name      string
		expected  interface{}
		actual    interface{}
		wantEqual bool
		wantDiffs int
	}{
		{
			name:      "Primitive Int (Numbers are float64 in JSON)",
			expected:  float64(42),
			actual:    float64(42),
			wantEqual: true,
			wantDiffs: 0,
		},
		{
			name:      "Primitive String",
			expected:  "hello",
			actual:    "hello",
			wantEqual: true,
			wantDiffs: 0,
		},
		{
			name:      "Primitive Mismatch type",
			expected:  "hello",
			actual:    float64(42),
			wantEqual: false,
			wantDiffs: 1,
		},
		{
			name:      "Primitive Mismatch value",
			expected:  float64(42),
			actual:    float64(43),
			wantEqual: false,
			wantDiffs: 1,
		},
		{
			name:      "Map Equal",
			expected:  map[string]interface{}{"a": float64(1), "b": "test"},
			actual:    map[string]interface{}{"a": float64(1), "b": "test"},
			wantEqual: true,
			wantDiffs: 0,
		},
		{
			name:      "Map Missing Key",
			expected:  map[string]interface{}{"a": float64(1), "b": "test"},
			actual:    map[string]interface{}{"a": float64(1)},
			wantEqual: false,
			wantDiffs: 1,
		},
		{
			name:      "Map Extra Key in Actual",
			expected:  map[string]interface{}{"a": float64(1)},
			actual:    map[string]interface{}{"a": float64(1), "b": "extra"},
			wantEqual: false,
			wantDiffs: 1,
		},
		{
			name:      "Array Equal",
			expected:  []interface{}{float64(1), float64(2), float64(3)},
			actual:    []interface{}{float64(1), float64(2), float64(3)},
			wantEqual: true,
			wantDiffs: 0,
		},
		{
			name:      "Array Length Mismatch",
			expected:  []interface{}{float64(1), float64(2)},
			actual:    []interface{}{float64(1)},
			wantEqual: false,
			wantDiffs: 1,
		},
		{
			name: "Nested Structure",
			expected: map[string]interface{}{
				"user": map[string]interface{}{"name": "Mario", "age": float64(30)},
			},
			actual: map[string]interface{}{
				"user": map[string]interface{}{"name": "Mario", "age": float64(30)},
			},
			wantEqual: true,
			wantDiffs: 0,
		},
		{
			name: "Nested Mismatch",
			expected: map[string]interface{}{
				"user": map[string]interface{}{"name": "Mario", "age": float64(30)},
			},
			actual: map[string]interface{}{
				"user": map[string]interface{}{"name": "Luigi", "age": float64(25)},
			},
			wantEqual: false,
			wantDiffs: 2,
		},
		{
			name: "Multiple diffs at different levels",
			expected: map[string]interface{}{
				"status": "active",
				"count":  float64(10),
				"nested": map[string]interface{}{"x": float64(1)},
			},
			actual: map[string]interface{}{
				"status": "inactive",
				"count":  float64(20),
				"nested": map[string]interface{}{"x": float64(99)},
			},
			wantEqual: false,
			wantDiffs: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedJSON, _ := json.Marshal(tt.expected)
			actualJSON, _ := json.Marshal(tt.actual)

			diffs, equal := DeepCompare(expectedJSON, actualJSON)

			if equal != tt.wantEqual {
				t.Errorf("DeepCompare() equal = %v, want %v", equal, tt.wantEqual)
			}

			if len(diffs) != tt.wantDiffs {
				t.Errorf("DeepCompare() returned %d diffs, want %d. Diffs: %v", len(diffs), tt.wantDiffs, diffs)
			}
		})
	}
}

func TestDeepCompareString(t *testing.T) {
	expected, _ := json.Marshal(map[string]interface{}{"a": float64(1)})
	actual, _ := json.Marshal(map[string]interface{}{"a": float64(2)})

	msg, equal := DeepCompareString(expected, actual)
	if equal {
		t.Error("expected inequality")
	}
	if msg == "" {
		t.Error("expected non-empty diff message")
	}
}

func TestFormatDiffs(t *testing.T) {
	diffs := []DiffEntry{
		{Path: "$.a", Message: "diff 1"},
		{Path: "$.b", Message: "diff 2"},
	}
	result := FormatDiffs(diffs)
	if result != "diff 1; diff 2" {
		t.Errorf("FormatDiffs() = %q, want %q", result, "diff 1; diff 2")
	}

	empty := FormatDiffs(nil)
	if empty != "" {
		t.Errorf("FormatDiffs(nil) = %q, want empty", empty)
	}
}
