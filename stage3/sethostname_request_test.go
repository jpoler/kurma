// Copyright 2013-2015 Apcera Inc. All rights reserved.

// +build linux,cgo

package stage3_test

import (
	"testing"
	"time"

	. "github.com/apcera/util/testtool"
)

func TestSetHostnameRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// Start the initd process.
	_, socket, _, _ := StartInitd(t)

	request := [][]string{[]string{"SETHOSTNAME", "test"}}
	reply, err := MakeRequest(socket, request, 2*time.Second)
	TestExpectSuccess(t, err)
	TestEqual(t, reply, "REQUEST OK\n")
}

func TestBadSetHostnameRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	tests := [][][]string{
		// Test 1: Request is too long.
		[][]string{
			[]string{"SETHOSTNAME", "testhost"},
			[]string{"EXTRA"},
		},

		// Test 2: Request is missing a hostname.
		[][]string{
			[]string{"SETHOSTNAME"},
		},

		// Test 3: Extra cruft.
		[][]string{
			[]string{"SETHOSTNAME", "testhost", "EXTRA"},
		},
	}
	BadResultsCheck(t, tests)
}
