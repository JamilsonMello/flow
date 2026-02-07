package flow

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func DeepCompare(expectedJSON, actualJSON json.RawMessage) (string, bool) {
	var expected, actual interface{}

	if err := json.Unmarshal(expectedJSON, &expected); err != nil {
		return fmt.Sprintf("failed to unmarshal expected: %v", err), false
	}
	if err := json.Unmarshal(actualJSON, &actual); err != nil {
		return fmt.Sprintf("failed to unmarshal actual: %v", err), false
	}

	return diff(expected, actual, "")
}

func diff(expected, actual interface{}, path string) (string, bool) {
	if expected == nil && actual == nil {
		return "", true
	}
	if expected == nil || actual == nil {
		return fmt.Sprintf("path %s: expected %v, got %v", path, expected, actual), false
	}

	v1 := reflect.ValueOf(expected)
	v2 := reflect.ValueOf(actual)

	if v1.Kind() != v2.Kind() {
		return fmt.Sprintf("path %s: type mismatch %T != %T", path, expected, actual), false
	}

	switch v1.Kind() {
	case reflect.Map:
		m1 := expected.(map[string]interface{})
		m2 := actual.(map[string]interface{})

		for k, val1 := range m1 {
			val2, ok := m2[k]
			if !ok {
				return fmt.Sprintf("path %s.%s: missing key in actual", path, k), false
			}
			if msg, ok := diff(val1, val2, path+"."+k); !ok {
				return msg, false
			}
		}

		for k := range m2 {
			if _, ok := m1[k]; !ok {
				return fmt.Sprintf("path %s.%s: unexpected key in actual", path, k), false
			}
		}
		return "", true

	case reflect.Slice:
		s1 := expected.([]interface{})
		s2 := actual.([]interface{})
		if len(s1) != len(s2) {
			return fmt.Sprintf("path %s: array length mismatch %d != %d", path, len(s1), len(s2)), false
		}
		for i := 0; i < len(s1); i++ {
			if msg, ok := diff(s1[i], s2[i], fmt.Sprintf("%s[%d]", path, i)); !ok {
				return msg, false
			}
		}
		return "", true

	default:
		if !reflect.DeepEqual(expected, actual) {
			return fmt.Sprintf("path %s: expected %v, got %v", path, expected, actual), false
		}
		return "", true
	}
}
