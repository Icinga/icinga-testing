package utils

import (
	"testing"
)

// MustString panics if err != nil and returns s otherwise.
func MustString(s string, err error) string {
	if err != nil {
		panic(err)
	} else {
		return s
	}
}

// mustT wraps a testing.TB to implement receiver functions on it.
type mustT struct {
	t testing.TB
}

// MustT returns a struct with receiver functions that fail the given testing.TB on errors.
//
// For example, MustT(t).String(fn()), where fn is of type func() (string, error), returns the string returned by
// fn if it did not return an error, or calls t.Fatal(err) otherwise.
func MustT(t testing.TB) mustT {
	return mustT{t: t}
}

// String calls t.Fatal(err) on the testing.TB passed to MustT if err != nil, otherwise it returns s.
func (m mustT) String(s string, err error) string {
	if err != nil {
		m.t.Fatal(err)
		return ""
	} else {
		return s
	}
}
