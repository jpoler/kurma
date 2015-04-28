// Copyright 2012-2013 Apcera Inc. All rights reserved.

package unittest_test

import (
	"fmt"

	"github.com/apcera/logray"
	"github.com/apcera/logray/unittest"
)

func Example() {
	// Setup
	buffer := unittest.SetupBuffer()
	defer buffer.DumpToStdout()

	// Unit tests go here.

	// Clear the buffer so nothing ends up being printed.
	buffer.Clear()
}

func ExamplePass() {
	// Setup
	buffer := unittest.SetupBuffer()
	defer buffer.DumpToStdout()

	// Log a bunch of stuff.
	logger := logray.New()
	fmt.Println("Expected output.")
	logger.Info("log line 1")
	logger.Info("log line 2")
	logger.Info("log line 3")

	// Clear the buffer so nothing ends up on stdout.
	buffer.Clear()

	// Output: Expected output.
}
