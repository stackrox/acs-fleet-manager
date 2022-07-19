package api

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// JSON ...
type JSON json.RawMessage

// Scan scan value into Jsonb, implements sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal json value: %v", value)
	}

	result := json.RawMessage{}
	err := json.Unmarshal(bytes, &result)
	*j = JSON(result)
	return err
}

// Value return json value, implement driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 || string(j) == "null" {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// MarshalJSON ...
func (m JSON) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *JSON) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("api.JSON: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

// Object ...
func (m JSON) Object() (map[string]interface{}, error) {
	if m == nil {
		return nil, nil
	}

	result := map[string]interface{}{}
	err := json.Unmarshal(m, &result)
	return result, err
}
