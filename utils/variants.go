package utils

import (
	"reflect"
)

type variants struct {
	base   reflect.Value
	result []interface{}
}

func MakeVariants(base interface{}) *variants {
	value := reflect.ValueOf(base)
	if value.Type().Kind() != reflect.Struct {
		panic("base must be of some struct type")
	}
	return &variants{base: value}
}

func (v *variants) Vary(field string, values ...interface{}) *variants {
	for _, value := range values {
		elem := reflect.New(v.base.Type()).Elem()           // elem := structType{}
		elem.Set(v.base)                                    // elem = base
		elem.FieldByName(field).Set(reflect.ValueOf(value)) // elem.Field = value
		v.result = append(v.result, elem.Interface())
	}
	return v
}

func (v *variants) Result() []interface{} {
	return v.result
}

func (v *variants) ResultAsBaseTypeSlice() interface{} {
	result := reflect.MakeSlice(reflect.SliceOf(v.base.Type()), len(v.result), len(v.result))
	for i, elem := range v.result {
		result.Index(i).Set(reflect.ValueOf(elem))
	}
	return result.Interface()
}
