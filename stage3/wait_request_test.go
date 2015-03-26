// Copyright 2013-2015 Apcera Inc. All rights reserved.

// +build linux,cgo

package stage3

import (
	"io/ioutil"
	"path"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	. "github.com/apcera/util/testtool"
)

func TestWaitRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// Start the initd process.
	cgroup, socket, log, pid := StartInitd(t)

	// Runs the wait call in the background in order to allow us to do some
	// work in the test and see if the wait routine has exited.
	var errWait error
	var replyWait string
	done := false
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() { done = true }()
		request := [][]string{[]string{"WAIT"}}
		replyWait, errWait = MakeRequest(socket, request, 10*time.Second)
	}()

	// Now make sure that the wait call was received and is.. erm.. waiting.
	Timeout(t, 10*time.Second, time.Second/10, func() bool {
		data, err := ioutil.ReadFile(log)
		TestExpectSuccess(t, err)
		return strings.Contains(string(data), "Added to the waiting queue.")
	})
	TestEqual(t, done, false)

	// Now start a process and ensure that the wait command does NOT terminate.
	dir := TempDir(t)
	stdout := path.Join(dir, "stdout")
	stderr := path.Join(dir, "stderr")
	request := [][]string{
		[]string{"START", "sleep"},
		[]string{"/bin/sleep", "60"},
		[]string{"KTEST=VTEST"},
		[]string{stdout, stderr},
		[]string{"99", "99"},
	}
	reply, err := MakeRequest(socket, request, 10*time.Second)
	TestExpectSuccess(t, err)
	TestEqual(t, reply, "REQUEST OK\n")

	// See that the sleep process has started.
	Timeout(t, 5*time.Second, time.Second/100, func() bool {
		tasks, err := cgroup.Tasks()
		TestExpectSuccess(t, err)
		return len(tasks) == 2
	})

	// Verify that the wait routine has not stopped.
	TestEqual(t, done, false)

	// And now, kill the sleep process and ensure that the wait routine
	// actually does stop.
	tasks, err := cgroup.Tasks()
	TestExpectSuccess(t, err)
	TestEqual(t, len(tasks), 2)
	if tasks[0] == pid {
		syscall.Kill(tasks[1], syscall.SIGKILL)
	} else {
		syscall.Kill(tasks[0], syscall.SIGKILL)
	}

	// Give the wait request 5 seconds to complete.
	Timeout(t, 5*time.Second, time.Second/100, func() bool {
		return done
	})

	// Check the results.
	TestExpectSuccess(t, errWait)
	TestEqual(t, replyWait, "REQUEST OK\n")
}

func TestBadWaitRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	tests := [][][]string{
		// Test 1: Request Extra cruft after STATUS
		[][]string{
			[]string{"WAIT", "EXTRA"},
		},

		// Test 2: Request is too long..
		[][]string{
			[]string{"WAIT"},
			[]string{},
		},
	}
	BadResultsCheck(t, tests)
}
