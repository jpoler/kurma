// Copyright 2013-2015 Apcera, Inc. All rights reserved.

// +build linux,cgo

package stage3

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/apcera/util/testtool"
)

func TestBadGeneralRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// Start the initd process.
	_, socket, _, _ := StartInitd(t)

	errors := make([]string, 0, 100)
	errorsMutex := sync.Mutex{}
	testWG := sync.WaitGroup{}

	// Adds an error to the array.
	addError := func(s string) {
		errorsMutex.Lock()
		defer errorsMutex.Unlock()
		errors = append(errors, s)
	}

	// Runs the given test in the background against the running initd server.
	runTest := func(name, s string) {
		testWG.Add(1)
		go func() {
			defer testWG.Done()
			reply, err := RawRequest(socket, s, time.Second*30)
			if err == nil {
				addError(fmt.Sprintf(
					"%s: Request succeeded when it shouldn't have.", name))
			} else if reply != "PROTOCOL ERROR\n" {
				addError(fmt.Sprintf("%s: Return was wrong: %s", name, reply))
			}
		}()
	}

	// Test 1: Bad protocol (0)
	runTest("test 1", "0\n")

	// Test 2: Bad protocol (2)
	runTest("test 2", "2\n")

	// Test 3: Invalid character in the protocol.
	runTest("test 3", "A\n")

	// Test 4: zero outer length.
	runTest("test 4", "1\n0\n")

	// Test 5: Excessively huge outer array.
	runTest("test 5", "1\n1000000000\n")

	// Test 6: Excessively huge inner array.
	runTest("test 6", "1\n1\n1000000000\n")

	// Test 7: Excessively huge string length.
	runTest("test 7", "1\n1\n1\n1000000000\n")

	// Test 8: Missing command.
	runTest("test 8", "1\n1\n0\n")

	// Test 9: Unknown command.
	runTest("test 9", "1\n1\n1\n7\nUNKNOWN")

	// Wait for the tests to finish up.
	testWG.Wait()
	if len(errors) != 0 {
		Fatalf(t, "Errors running tests:\n%s", strings.Join(errors, "\n"))
	}
}
