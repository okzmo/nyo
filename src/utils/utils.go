package utils

import (
	"fmt"
	"strings"
)

func ContainsSubstring(s string, keywords []string) (string, bool) {
	for _, keyword := range keywords {
		if strings.Contains(s, keyword) {
			return keyword, true
		}
	}

	return "", false
}

func ConvertToStringSlice(input any) ([]string, error) {
	slice, ok := input.([]any)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", input)
	}

	result := make([]string, len(slice))
	for i, v := range slice {
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("element at index %d is not a string: %v", i, v)
		}
		result[i] = str
	}

	return result, nil
}
