package utils

import (
	"reflect"
	"testing"
)

func TestMakeVariants(t *testing.T) {
	type T struct {
		I int
		B bool
		S string
		X int
	}

	base := T{I: -1, B: true, S: "default", X: 1337}

	got := MakeVariants(base).
		Vary("I", 23, 42).
		Vary("B", true, false).
		Vary("S", "foo", "bar", "baz").
		Result()

	want := []interface{}{
		// Base
		base,

		// I: {23, 42}
		T{I: 23, B: true, S: "default", X: 1337},
		T{I: 42, B: true, S: "default", X: 1337},

		// B: {true, false}
		T{I: -1, B: true, S: "default", X: 1337},
		T{I: -1, B: false, S: "default", X: 1337},

		// S: {"foo", "bar", "baz"}
		T{I: -1, B: true, S: "foo", X: 1337},
		T{I: -1, B: true, S: "bar", X: 1337},
		T{I: -1, B: true, S: "baz", X: 1337},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("variants() = %v, want %v", got, want)
	}
}

func TestMakeVariantsAsBaseTypeSlice(t *testing.T) {
	type T struct {
		A int
		B int
	}

	got := MakeVariants(T{}).
		Vary("A", 23, 42).
		Vary("B", 1337).
		ResultAsBaseTypeSlice().([]T)

	want := []T{
		// Base
		{},

		// A: {23, 42}
		{A: 23},
		{A: 42},

		// B: {1337}
		{B: 1337},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("variants() = %v, want %v", got, want)
	}
}

func TestVariantInfoString(t *testing.T) {
	type T struct {
		A int
		B int
		VariantInfo
	}

	variants := MakeVariants(T{}).
		Vary("A", 23, 42).
		Vary("B", 1337).
		ResultAsBaseTypeSlice().([]T)

	got := make([]string, 0, len(variants))
	for _, variant := range variants {
		got = append(got, variant.VariantInfoString())
	}

	want := []string{
		// Base
		"Base",

		// A: {23, 42}
		"A-1",
		"A-2",

		// B: {1337}
		"B-1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("variants() = %v, want %v", got, want)
	}
}
