// Copyright 2013-2015 Apcera Inc. All rights reserved.

// +build linux,cgo

package stage3_test

import (
	"io/ioutil"
	"strings"
	"testing"
	"time"

	. "github.com/apcera/util/testtool"
)

func TestStartup(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// Start the initd process.
	cgroup, _, log, _ := StartInitd(t)

	// Check the process tree to see that there is exactly one process
	// with no children. This Ensures that golang did not startup by
	// checking that only one process exists in the process group.
	tasks, err := cgroup.Tasks()
	TestExpectSuccess(t, err)
	TestEqual(t, len(tasks), 2)

	// Ensure that the socket file is opened.
	Timeout(t, time.Second, time.Second/100, func() bool {
		data, err := ioutil.ReadFile(log)
		TestExpectSuccess(t, err)
		return strings.Contains(string(data), "Socket file")
	})

	// Ensure that the signal handler was setup via the logs.
	Timeout(t, time.Second, time.Second/100, func() bool {
		data, err := ioutil.ReadFile(log)
		TestExpectSuccess(t, err)
		return strings.Contains(string(data), "Setup signal handler.")
	})

	// Ensure that the logs file contains the string showing us that it has
	// started properly.
	Timeout(t, time.Second, time.Second/100, func() bool {
		data, err := ioutil.ReadFile(log)
		TestExpectSuccess(t, err)
		return strings.Contains(string(data), "Starting initd.")
	})
}
