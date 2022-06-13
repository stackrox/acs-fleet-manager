package util

import (
	"crypto/md5"
	"encoding/json"
)

func MD5SumFromJSONStruct(in interface{}) ([16]byte, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return [16]byte{}, err
	}

	return md5.Sum(b), nil
}
