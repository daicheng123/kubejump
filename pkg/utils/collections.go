package utils

import "reflect"

// IsZero 校验是否为0值
func IsZero(v interface{}) bool {
	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	return value.IsZero()
}
