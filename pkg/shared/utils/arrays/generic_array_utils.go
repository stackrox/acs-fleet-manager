// Package arrays ...
package arrays

// FindFirst Finds the first element into the passed in values where the predicate is true
// WARNING: passed in values are converted to an []interface{} slice and returned value will be an interface{}
func FindFirst(predicate func(x interface{}) bool, values ...interface{}) (int, interface{}) {
	for i, val := range values {
		if predicate(val) {
			return i, val
		}
	}
	return -1, nil
}
