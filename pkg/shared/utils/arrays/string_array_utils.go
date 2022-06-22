package arrays

import "fmt"

// FindFirstString Finds the first element into the passed in values where the predicate is true
// Returns: -1 and empty string when not found, position and the string itself otherwise.
func FindFirstString(values []string, predicate func(x string) bool) int {
	for i, val := range values {
		if predicate(val) {
			return i
		}
	}
	return -1
}

// FilterStringSlice return a slices containing only the values of `values` matching the given predicate
func FilterStringSlice(values []string, predicate func(x string) bool) []string {
	var res []string
	for _, val := range values {
		if predicate(val) {
			res = append(res, val)
		}
	}
	return res
}

// FirstNonEmpty Returns the first non-empty string between the passed in values.
// If no non-empty string can be found, returns an empty string and an error
func FirstNonEmpty(values ...string) (string, error) {
	if idx := FindFirstString(values, func(s string) bool { return s != "" }); idx == -1 {
		return "", fmt.Errorf("all strings are empty")
	} else {
		return values[idx], nil
	}
}

// FirstNonEmptyOrDefault Returns the first non-empty string between the passed in values.
// If no non-empty string can be found, returns defaultValue
func FirstNonEmptyOrDefault(defaultValue string, values ...string) string {
	if idx := FindFirstString(values, func(s string) bool { return s != "" }); idx == -1 {
		return defaultValue
	} else {
		return values[idx]
	}
}

func Contains(values []string, s string) bool {
	return FindFirstString(values, func(x string) bool { return x == s }) != -1
}
