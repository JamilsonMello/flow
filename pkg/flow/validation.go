package flow

import (
	"encoding/json"
)

func ValidateWithSchema(expected, actual, schema json.RawMessage) (string, bool) {
	var expectedVal, actualVal interface{}

	if err := json.Unmarshal(expected, &expectedVal); err != nil {
		return "Invalid expected value: " + err.Error(), false
	}

	if err := json.Unmarshal(actual, &actualVal); err != nil {
		return "Invalid actual value: " + err.Error(), false
	}

	if expectedVal != nil && actualVal != nil {
		if err := validateTypes(expectedVal, actualVal); err != nil {
			return "Type mismatch: " + err.Error(), false
		}
	}

	return DeepCompareString(expected, actual)
}

func validateTypes(expected, actual interface{}) error {
	switch expected.(type) {
	case string:
		if _, ok := actual.(string); !ok {
			return &TypeError{expected: "string", actual: actual}
		}
	case float64:
		if _, ok := actual.(float64); !ok {
			return &TypeError{expected: "number", actual: actual}
		}
	case bool:
		if _, ok := actual.(bool); !ok {
			return &TypeError{expected: "boolean", actual: actual}
		}
	case []interface{}:
		if _, ok := actual.([]interface{}); !ok {
			return &TypeError{expected: "array", actual: actual}
		}
	case map[string]interface{}:
		if _, ok := actual.(map[string]interface{}); !ok {
			return &TypeError{expected: "object", actual: actual}
		}
	}
	return nil
}

type TypeError struct {
	expected string
	actual   interface{}
}

func (e *TypeError) Error() string {
	return "expected " + e.expected + ", got " + getTypeName(e.actual)
}

func getTypeName(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}
