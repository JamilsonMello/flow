package flow

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type DiffEntry struct {
	Path     string      `json:"path"`
	Expected interface{} `json:"expected"`
	Actual   interface{} `json:"actual"`
	Message  string      `json:"message"`
}

func DeepCompare(expectedJSON, actualJSON json.RawMessage) ([]DiffEntry, bool) {
	var expected, actual interface{}

	if err := json.Unmarshal(expectedJSON, &expected); err != nil {
		return []DiffEntry{{Path: "$", Message: fmt.Sprintf("failed to unmarshal expected: %v", err)}}, false
	}
	if err := json.Unmarshal(actualJSON, &actual); err != nil {
		return []DiffEntry{{Path: "$", Message: fmt.Sprintf("failed to unmarshal actual: %v", err)}}, false
	}

	var diffs []DiffEntry
	collectDiffs(expected, actual, "$", &diffs)
	return diffs, len(diffs) == 0
}

func FormatDiffs(diffs []DiffEntry) string {
	if len(diffs) == 0 {
		return ""
	}
	var parts []string
	for _, d := range diffs {
		parts = append(parts, d.Message)
	}
	return strings.Join(parts, "; ")
}

func DeepCompareString(expectedJSON, actualJSON json.RawMessage) (string, bool) {
	diffs, equal := DeepCompare(expectedJSON, actualJSON)
	return FormatDiffs(diffs), equal
}

func collectDiffs(expected, actual interface{}, path string, diffs *[]DiffEntry) {
	if expected == nil && actual == nil {
		return
	}
	if expected == nil || actual == nil {
		*diffs = append(*diffs, DiffEntry{
			Path:     path,
			Expected: expected,
			Actual:   actual,
			Message:  fmt.Sprintf("path %s: expected %v, got %v", path, expected, actual),
		})
		return
	}

	v1 := reflect.ValueOf(expected)
	v2 := reflect.ValueOf(actual)

	if v1.Type() != v2.Type() {
		*diffs = append(*diffs, DiffEntry{
			Path:     path,
			Expected: expected,
			Actual:   actual,
			Message:  fmt.Sprintf("path %s: type mismatch expected %T, got %T", path, expected, actual),
		})
		return
	}

	switch v1.Kind() {
	case reflect.Map:
		m1, ok1 := expected.(map[string]interface{})
		m2, ok2 := actual.(map[string]interface{})
		if !ok1 || !ok2 {
			*diffs = append(*diffs, DiffEntry{
				Path:    path,
				Message: fmt.Sprintf("path %s: expected map, got %T", path, actual),
			})
			return
		}

		for k, val1 := range m1 {
			val2, ok := m2[k]
			if !ok {
				*diffs = append(*diffs, DiffEntry{
					Path:     path + "." + k,
					Expected: val1,
					Actual:   nil,
					Message:  fmt.Sprintf("path %s.%s: key missing in actual", path, k),
				})
				continue
			}
			collectDiffs(val1, val2, path+"."+k, diffs)
		}

		for k, val2 := range m2 {
			if _, ok := m1[k]; !ok {
				*diffs = append(*diffs, DiffEntry{
					Path:     path + "." + k,
					Expected: nil,
					Actual:   val2,
					Message:  fmt.Sprintf("path %s.%s: unexpected extra key in actual", path, k),
				})
			}
		}

	case reflect.Slice:
		s1, ok1 := expected.([]interface{})
		s2, ok2 := actual.([]interface{})
		if !ok1 || !ok2 {
			*diffs = append(*diffs, DiffEntry{
				Path:    path,
				Message: fmt.Sprintf("path %s: expected array, got %T", path, actual),
			})
			return
		}

		if len(s1) != len(s2) {
			*diffs = append(*diffs, DiffEntry{
				Path:     path,
				Expected: len(s1),
				Actual:   len(s2),
				Message:  fmt.Sprintf("path %s: array length mismatch %d != %d", path, len(s1), len(s2)),
			})
			return
		}

		for i := 0; i < len(s1); i++ {
			collectDiffs(s1[i], s2[i], fmt.Sprintf("%s[%d]", path, i), diffs)
		}

	default:
		if !reflect.DeepEqual(expected, actual) {
			*diffs = append(*diffs, DiffEntry{
				Path:     path,
				Expected: expected,
				Actual:   actual,
				Message:  fmt.Sprintf("path %s: value mismatch expected %v, got %v", path, expected, actual),
			})
		}
	}
}
