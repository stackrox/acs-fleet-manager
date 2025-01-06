package db

import (
	"reflect"
)

// IsStructWithoutPointers checks that a struct has no pointer members. This is
// to ensure a DB model can be passed to the gorm Update() method and all fields
// will be taken into account.
func IsStructWithoutPointers(s any) bool {
	val := reflect.Indirect(reflect.ValueOf(s))
	typ := val.Type()

	if typ.Kind() != reflect.Struct {
		return false
	}

	for i := range typ.NumField() {
		if typ.Field(i).Type.Kind() == reflect.Ptr {
			return false
		}
	}

	return true
}
