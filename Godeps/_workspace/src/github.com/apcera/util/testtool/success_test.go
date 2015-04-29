// Copyright 2014 Apcera Inc. All rights reserved.

package testtool

import (
	"errors"
	"testing"
)

type SimpleError struct {
	Msg string
}

func (se *SimpleError) Error() string { return se.Msg }

func TestTestExpectSuccess(t *testing.T) {
	m := &MockLogger{}

	var e1nil error
	e2foo := errors.New("foo")
	s3foo := &SimpleError{Msg: "foo"}
	var s4nil *SimpleError

	m.RunTest(t, false, func() { TestExpectSuccess(m, nil, "simple nil") })
	m.RunTest(t, false, func() { TestExpectSuccess(m, e1nil, "nil via interface var") })
	m.RunTest(t, true, func() { TestExpectSuccess(m, e2foo, "non-nil via interface var") })
	m.RunTest(t, true, func() { TestExpectSuccess(m, s3foo, "non-nil via concrete var") })
	m.RunTest(t, false, func() { TestExpectSuccess(m, s4nil, "nil via concrete var") })
}
