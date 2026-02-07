package flow

import (
	"encoding/json"
	"testing"
)

func TestDeepCompare(t *testing.T) {
	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
		isValid  bool
	}{
		{
			name:     "Primitive Int (Numbers are float64 in JSON)",
			expected: 10.0, // Pre-converted
			actual:   10.0,
			isValid:  true,
		},
		{
			name:     "Primitive String",
			expected: "hello",
			actual:   "hello",
			isValid:  true,
		},
		{
			name:     "Primitive Mismatch type",
			expected: 10.0,
			actual:   "10",
			isValid:  false,
		},
		{
			name:     "Primitive Mismatch value",
			expected: 10.0,
			actual:   20.0,
			isValid:  false,
		},
		{
			name:     "Map Equal",
			expected: map[string]interface{}{"a": 1.0},
			actual:   map[string]interface{}{"a": 1.0},
			isValid:  true,
		},
		{
			name:     "Map Missing Key",
			expected: map[string]interface{}{"a": 1.0, "b": 2.0},
			actual:   map[string]interface{}{"a": 1.0},
			isValid:  false,
		},
		{
			name:     "Array Equal",
			expected: []interface{}{1.0, 2.0},
			actual:   []interface{}{1.0, 2.0},
			isValid:  true,
		},
		{
			name:     "Array Length Mismatch",
			expected: []interface{}{1.0},
			actual:   []interface{}{1.0, 2.0},
			isValid:  false,
		},
		{
			name:     "Nested Structure",
			expected: map[string]interface{}{"user": map[string]interface{}{"id": 1.0}},
			actual:   map[string]interface{}{"user": map[string]interface{}{"id": 1.0}},
			isValid:  true,
		},
		{
			name:     "Nested Mismatch",
			expected: map[string]interface{}{"user": map[string]interface{}{"id": 1.0}},
			actual:   map[string]interface{}{"user": map[string]interface{}{"id": 2.0}},
			isValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expBytes, _ := json.Marshal(tt.expected)
			actBytes, _ := json.Marshal(tt.actual)

			msg, ok := DeepCompare(expBytes, actBytes)
			if ok != tt.isValid {
				t.Errorf("DeepCompare() valid = %v, want %v. Msg: %s", ok, tt.isValid, msg)
			}
		})
	}
}
