package output

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// ApplyJQ applies a jq expression to data and returns the result.
func ApplyJQ(data interface{}, expr string) (interface{}, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("jq parse: %w", err)
	}

	normalized, err := normalizeForJQ(data)
	if err != nil {
		return nil, fmt.Errorf("jq normalize: %w", err)
	}

	iter := query.Run(normalized)
	var results []interface{}
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return nil, fmt.Errorf("jq eval: %w", err)
		}
		results = append(results, v)
	}

	if len(results) == 0 {
		return nil, nil
	}
	if len(results) == 1 {
		return results[0], nil
	}
	return results, nil
}

func normalizeForJQ(data interface{}) (interface{}, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var normalized interface{}
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}
