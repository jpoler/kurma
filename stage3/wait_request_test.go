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

	// Start a process with a start command that sleeps for
	// some time. This way we can test the blocking behavior of
	// WAIT RPC.
	dir := TempDir(t)
	stdout := path.Join(dir, "stdout")
	stderr := path.Join(dir, "stderr")
	request := [][]string{
		[]string{"START", "sleep"},
		[]string{"/bin/sleep", "60"},
		[]string{""},
		[]string{"KTEST=VTEST"},
		[]string{stdout, stderr},
		[]string{"99", "99"},
	}
	reply, err := MakeRequest(socket, request, 10*time.Second)
	TestExpectSuccess(t, err)
	TestEqual(t, reply, "REQUEST OK\n")

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

	// Now make sure that the wait call was received and blocked on
	// waiting for the process.
	Timeout(t, 10*time.Second, time.Second/10, func() bool {
		data, err := ioutil.ReadFile(log)
		TestExpectSuccess(t, err)
		return strings.Contains(string(data), "Added to the waiting queue")
	})
	TestEqual(t, done, false)

	// See that the sleep process has started.
	Timeout(t, 5*time.Second, time.Second/100, func() bool {
		tasks, err := cgroup.Tasks()
		TestExpectSuccess(t, err)
		return len(tasks) == 2
	})

	// Verify that the wait routine has not stopped.
	TestEqual(t, done, false)

	// And now, kill the sleep process and ensure that the wait
	// routine actually does stop.
	tasks, err := cgroup.Tasks()
	TestExpectSuccess(t, err)
	TestEqual(t, len(tasks), 2)
	if tasks[0] == pid {
		syscall.Kill(tasks[1], syscall.SIGKILL)
	} else {
		syscall.Kill(tasks[0], syscall.SIGKILL)
	}

	Timeout(t, 5*time.Second, time.Second/100, func() bool {
		tasks, err := cgroup.Tasks()
		TestExpectSuccess(t, err)
		return len(tasks) == 1
	})

	wg.Wait()

	// Check the results.
	TestExpectSuccess(t, errWait)
	TestEqual(t, replyWait, "REQUEST OK\n")
}

func TestWaitRequestDoesntBlockProcessesAreFinished(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// Start the initd process.
	cgroup, socket, log, _ := StartInitd(t)

	// Start a process with a start command that sleeps for a very
	// short period of time, so we can issue WAIT after it's done and
	// expect it not to block.
	dir := TempDir(t)
	stdout := path.Join(dir, "stdout")
	stderr := path.Join(dir, "stderr")
	request := [][]string{
		[]string{"START", "sleep"},
		[]string{"/bin/sleep", "0.05"},
		[]string{""},
		[]string{"KTEST=VTEST"},
		[]string{stdout, stderr},
		[]string{"99", "99"},
	}
	reply, err := MakeRequest(socket, request, 10*time.Second)
	TestExpectSuccess(t, err)
	TestEqual(t, reply, "REQUEST OK\n")

	time.Sleep(200 * time.Millisecond)

	tasks, err := cgroup.Tasks()
	TestExpectSuccess(t, err)
	TestEqual(t, len(tasks), 1)

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

	// Now make sure that the wait call was received and answered
	// immediately, not blocked on the process to finish.
	Timeout(t, 10*time.Second, time.Second/10, func() bool {
		data, err := ioutil.ReadFile(log)
		TestExpectSuccess(t, err)
		return strings.Contains(string(data), "responding to WAIT immediately")
	})
	TestEqual(t, done, true)
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
