package defaults

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PrettyPrintDefaults returns a slice of human-readable lines (e.g. for logging)
// of the provided object marshalled as JSON.
func PrettyPrintDefaults(obj interface{}, label string) ([]string, error) {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("JSON marshalling of default resource settings for %s failed: %w", label, err)
	}

	lines := append(
		[]string{fmt.Sprintf("%s:", label)},
		strings.Split(string(bytes), "\n")...,
	)
	return lines, nil
}
