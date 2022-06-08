package util

import (
	"crypto/md5"
	"encoding/json"
)

func MD5SumFromJSONStruct(in interface{}) ([16]byte, error) {
	var sum [16]byte

	b, err := json.Marshal(in)
	if err != nil {
		return sum, err
	}

	sum = md5.Sum(b)
	return sum, nil
}
