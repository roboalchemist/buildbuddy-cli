package output

import (
	"encoding/json"
	"strings"
)

// ParseFieldList parses a comma-separated field list string.
func ParseFieldList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	fields := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			fields = append(fields, p)
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// FilterFields returns only the specified fields from data.
// Field names are matched against JSON keys. Supports dot notation for nesting.
// If data is a slice, applies filtering to each element.
func FilterFields(data interface{}, fields []string) interface{} {
	if len(fields) == 0 {
		return data
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return data
	}

	var generic interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		return data
	}

	return filterValue(generic, fields)
}

func filterValue(v interface{}, fields []string) interface{} {
	switch val := v.(type) {
	case []interface{}:
		result := make([]interface{}, 0, len(val))
		for _, item := range val {
			result = append(result, filterValue(item, fields))
		}
		return result
	case map[string]interface{}:
		return filterMap(val, fields)
	default:
		return v
	}
}

func filterMap(m map[string]interface{}, fields []string) map[string]interface{} {
	result := make(map[string]interface{})

	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		parts := strings.SplitN(field, ".", 2)
		topKey := parts[0]

		val, exists := m[topKey]
		if !exists {
			continue
		}

		if len(parts) == 1 {
			result[topKey] = val
		} else {
			childField := parts[1]
			childMap, ok := val.(map[string]interface{})
			if !ok {
				continue
			}
			existing, exists := result[topKey]
			if exists {
				if existingMap, ok := existing.(map[string]interface{}); ok {
					merged := filterMap(childMap, []string{childField})
					for k, v := range merged {
						existingMap[k] = v
					}
					result[topKey] = existingMap
					continue
				}
			}
			result[topKey] = filterMap(childMap, []string{childField})
		}
	}

	return result
}
