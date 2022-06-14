package util

import (
	"crypto/md5"
	"encoding/json"
)

// MD5SumFromJSONStruct return md5.Sum of given input marshaled to JSON
func MD5SumFromJSONStruct(in interface{}) ([16]byte, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return [16]byte{}, err
	}

	return md5.Sum(b), nil
}
