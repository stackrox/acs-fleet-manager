package handlers

import (
	"reflect"
)

// Prepare a 'list' of non-db-backed resources
func DetermineListRange(obj interface{}, page int, size int) (list []interface{}, total int) {
	items := reflect.ValueOf(obj)
	total = items.Len()
	low := (page - 1) * size
	high := low + size
	if low < 0 || low >= total || high >= total {
		low = 0
		high = total
	}
	for i := low; i < high; i++ {
		list = append(list, items.Index(i).Interface())
	}

	return list, total
}
