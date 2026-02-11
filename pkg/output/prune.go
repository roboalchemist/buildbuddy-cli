package output

import "strings"

// PruneEmpty recursively removes null, empty strings, whitespace-only strings,
// empty arrays, and empty objects from a JSON-compatible value.
func PruneEmpty(v interface{}) interface{} {
	result := pruneEmpty(v)
	if result == nil {
		return map[string]interface{}{}
	}
	return result
}

func pruneEmpty(v interface{}) interface{} {
	switch val := v.(type) {
	case nil:
		return nil
	case map[string]interface{}:
		pruned := make(map[string]interface{})
		for k, child := range val {
			if result := pruneEmpty(child); result != nil {
				pruned[k] = result
			}
		}
		if len(pruned) == 0 {
			return nil
		}
		return pruned
	case []interface{}:
		var pruned []interface{}
		for _, child := range val {
			if result := pruneEmpty(child); result != nil {
				pruned = append(pruned, result)
			}
		}
		if len(pruned) == 0 {
			return nil
		}
		return pruned
	case string:
		if strings.TrimSpace(val) == "" {
			return nil
		}
		return val
	case float64:
		return val
	case bool:
		return val
	default:
		return val
	}
}
