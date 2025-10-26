package util

import "reflect"

func IsSlice(value interface{}) bool {
	kind := reflect.TypeOf(value).Kind()

	if kind == reflect.Pointer {
		kind = reflect.TypeOf(value).Elem().Kind()
	}

	return kind == reflect.Slice
}
