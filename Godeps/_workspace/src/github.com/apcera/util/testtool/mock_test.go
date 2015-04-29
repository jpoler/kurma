// Copyright 2013 Apcera Inc. All rights reserved.

package testtool

import (
	"runtime"
	"sync"
	"testing"
)

type MockLogger struct {
	// This is set to true if Fatal or Fatalf have been called.
	failed bool

	// This is set to true if Skip or Skipf have been called.
	skipped bool

	// These functions will be called if set to non nil.
	funcError  func(args ...interface{})
	funcErrorf func(format string, args ...interface{})
	funcFailed func() bool
	funcFatal  func(args ...interface{})
	funcFatalf func(format string, args ...interface{})
	funcSkip   func(args ...interface{})
	funcSkipf  func(format string, args ...interface{})
}

func (m *MockLogger) RunTest(t *testing.T, fails bool, f func()) {
	m.failed = false
	m.skipped = false

	// We run the test in another goroutine since Fatal terminates
	// running goroutines.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
	wg.Wait()
	_, file, line, _ := runtime.Caller(1)
	if fails && !m.failed {
		t.Fatalf("\n%s:%d\nTest didn't fail but should have.", file, line)
	} else if !fails && m.failed {
		t.Fatalf("\n%s:%d\nTest failed but shouldn't have.", file, line)
	}
}

func (m *MockLogger) Error(args ...interface{}) {
	m.failed = true
	if m.funcError != nil {
		m.funcError(args...)
	}
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.failed = true
	if m.funcErrorf != nil {
		m.funcErrorf(format, args...)
	}
}

func (m *MockLogger) Failed() bool {
	if m.funcFailed != nil {
		return m.funcFailed()
	}
	return m.failed
}

func (m *MockLogger) Fatal(args ...interface{}) {
	m.failed = true
	if m.funcFatal != nil {
		m.funcFatal(args...)
	}
	runtime.Goexit()
}

func (m *MockLogger) Fatalf(format string, args ...interface{}) {
	m.failed = true
	if m.funcFatalf != nil {
		m.funcFatalf(format, args...)
	}
	runtime.Goexit()
}

func (m *MockLogger) Skip(args ...interface{}) {
	m.skipped = true
	if m.funcSkip != nil {
		m.funcSkip(args...)
	}
	runtime.Goexit()
}

func (m *MockLogger) Skipf(format string, args ...interface{}) {
	m.skipped = true
	if m.funcSkipf != nil {
		m.funcSkipf(format, args...)
	}
	runtime.Goexit()
}

func (m *MockLogger) Log(args ...interface{}) {
}

func (m *MockLogger) Logf(format string, args ...interface{}) {
}
