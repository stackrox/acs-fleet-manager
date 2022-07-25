package api

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
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
	if err != nil {
		return fmt.Errorf("unmarshalling json: %w", err)
	}
	return nil
}

// Value return json value, implement driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 || string(j) == "null" {
		return nil, nil
	}

	v, err := json.RawMessage(j).MarshalJSON()
	if err != nil {
		return v, fmt.Errorf("marshalling json: %w", err)
	}
	return v, nil
}

// MarshalJSON ...
func (j JSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("api.JSON: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// Object ...
func (j JSON) Object() (map[string]interface{}, error) {
	if j == nil {
		return nil, nil
	}

	result := map[string]interface{}{}
	err := json.Unmarshal(j, &result)
	if err != nil {
		return nil, fmt.Errorf("marshalling json: %w", err)
	}
	return result, nil
}
