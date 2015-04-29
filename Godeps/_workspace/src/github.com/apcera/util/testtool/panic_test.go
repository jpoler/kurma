// Copyright 2013 Apcera Inc. All rights reserved.

package testtool

import (
	"testing"
)

func TestTestExpectPanic(t *testing.T) {
	m := &MockLogger{}

	m.RunTest(t, false, func() { TestExpectPanic(m, func() { panic("Oh No!") }, "Oh No!") })
	m.RunTest(t, true, func() { TestExpectPanic(m, func() { panic("Oh No!") }, "Not Me") })
	m.RunTest(t, true, func() { TestExpectPanic(m, func() {}, "Oh No!") })
}
