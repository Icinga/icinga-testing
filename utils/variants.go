package utils

import (
	"fmt"
	"reflect"
)

type variants struct {
	base   reflect.Value
	result []interface{}
}

type VariantInfo struct {
	Field string
	Index int
	Value interface{}
}

func (v *VariantInfo) VariantInfoString() string {
	if v != nil && *v != (VariantInfo{}) {
		return fmt.Sprintf("%s-%d", v.Field, v.Index)
	} else {
		return "Base"
	}
}

type VariantInfoSetter interface {
	SetVariantInfo(field string, index int, value interface{})
}

func (v *VariantInfo) SetVariantInfo(field string, index int, value interface{}) {
	v.Field = field
	v.Index = index
	v.Value = value
}

func MakeVariants(base interface{}) *variants {
	value := reflect.ValueOf(base)
	if value.Type().Kind() != reflect.Struct {
		panic("base must be of some struct type")
	}
	return &variants{
		base:   value,
		result: []interface{}{base},
	}
}

func (v *variants) Vary(field string, values ...interface{}) *variants {
	for i, value := range values {
		elem := reflect.New(v.base.Type()).Elem()           // elem := structType{}
		elem.Set(v.base)                                    // elem = base
		elem.FieldByName(field).Set(reflect.ValueOf(value)) // elem.Field = value
		if e, ok := elem.Addr().Interface().(VariantInfoSetter); ok {
			e.SetVariantInfo(field, i+1, value)
		}
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
