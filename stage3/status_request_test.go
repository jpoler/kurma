// Copyright 2013-2015 Apcera Inc. All rights reserved.

// +build linux,cgo

package stage3

import (
	"path"
	"strings"
	"syscall"
	"testing"
	"time"

	. "github.com/apcera/util/testtool"
)

func TestStatusRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// Start the initd process.
	cgroup, socket, _, pid := StartInitd(t)

	// Process tracking.
	processes := make(map[string]int)
	pidnames := make(map[int]string)
	pidnames[pid] = "initd"
	processes["initd"] = pid

	// A directory that we can use to drop stuff in.
	dir := TempDir(t)

	// Starts a command with the given command line.
	startCmd := func(name string, cmd []string, longLived bool) {
		// Make a request against the initd server.
		stdout := path.Join(dir, "stdout-"+name)
		stderr := path.Join(dir, "stderr-"+name)
		request := [][]string{
			[]string{"START", name},
			cmd,
			[]string{"KTEST=VTEST"},
			[]string{stdout, stderr},
			[]string{"99", "99"},
		}
		reply, err := MakeRequest(socket, request, 10*time.Second)
		TestExpectSuccess(t, err)
		TestEqual(t, reply, "REQUEST OK\n")

		// Get the list of all tasks in the cgroup.
		var tasks []int
		if longLived {
			_, tasks = waitTask(t, cgroup, cmd, 5*time.Second)
		} else {
			// Still wait a little bit to allow initd to fork/exec
			time.Sleep(20 * time.Millisecond)
			tasks, err = cgroup.Tasks()
			TestExpectSuccess(t, err)
		}

		// Find the new pid that we do not know about.
		for _, pid := range tasks {
			if _, found := pidnames[pid]; !found {
				pidnames[pid] = name
				processes[name] = pid
				return
			}
		}

		if longLived {
			Fatalf(t, "Long lived process %s not found.", name)
		} else {
			processes[name] = -1
		}
	}

	// Reads the status from the processes and returns it as a map of
	// name:status
	getStatus := func() map[string]string {
		r := make(map[string]string, len(processes))
		request := [][]string{[]string{"STATUS"}}
		reply, err := MakeRequest(socket, request, 10*time.Second)
		TestExpectSuccess(t, err)
		replyStrs := strings.Split(reply, "\n")
		// There are three expected elements on every reply.
		// [0]: "REQUEST OK", [-2]: "END", and [-1]: ""
		// The "" is because the request terminates with a return which
		// causes replyStrs to split it to an empty string.
		if len(replyStrs) < 3 {
			Fatalf(t,
				"Length of reply elements is too short: %d: %#v",
				len(replyStrs), replyStrs)
		}
		TestEqual(t, replyStrs[0], "REQUEST OK")
		TestEqual(t, replyStrs[len(replyStrs)-2], "END")
		TestEqual(t, replyStrs[len(replyStrs)-1], "")

		// Now verity the remaining strings in the middle are a multiple
		// of two.
		remaining := replyStrs[1 : len(replyStrs)-2]
		TestEqual(t, len(remaining)%2, 0)

		// Walk through the tuples adding them to the array.
		for i := 0; i < len(remaining); i += 2 {
			r[remaining[i]] = remaining[i+1]
		}

		return r
	}

	// Ensure that the status request returns no processes initially.
	TestEqual(t, getStatus(), map[string]string{})

	// Start a few sleep commands to see if we can get the status from them.
	startCmd("sleep1", []string{"/bin/sleep", "61"}, true)
	startCmd("sleep2", []string{"/bin/sleep", "62"}, true)
	startCmd("sleep3", []string{"/bin/sleep", "63"}, true)
	TestEqual(t, getStatus(), map[string]string{
		"sleep1": "running",
		"sleep2": "running",
		"sleep3": "running",
	})

	// Now start a couple of /bin/true processes so we can see a status
	// switch to a successful or failed exit.
	startCmd("true", []string{"/bin/true"}, false)
	startCmd("false", []string{"/bin/false"}, false)

	// Wait for start and exit of these two commands.
	time.Sleep(50 * time.Millisecond)

	// Expect 4 processes (sleep x 3 and initd) since true and false should be
	// done executing.
	Timeout(t, 2*time.Second, 100*time.Millisecond, func() bool {
		tasks, err := cgroup.Tasks()
		TestExpectSuccess(t, err)
		return len(tasks) == 4
	})

	// Verify the exit status are valid.
	TestEqual(t, getStatus(), map[string]string{
		"sleep1": "running",
		"sleep2": "running",
		"sleep3": "running",
		"true":   "exited(0)",
		"false":  "exited(1)",
	})

	// Now kill two of the three sleeps and make sure that their status is
	// updated properly. (Note that FindProcess can not error on linux)
	syscall.Kill(processes["sleep1"], syscall.SIGTERM)
	syscall.Kill(processes["sleep2"], syscall.SIGKILL)

	Timeout(t, time.Second, 100*time.Millisecond, func() bool {
		// Expect 2 processes (sleep x 1 and initd)
		tasks, err := cgroup.Tasks()
		TestExpectSuccess(t, err)
		return len(tasks) == 2
	})

	// Verify the exit statuses are correct.
	TestEqual(t, getStatus(), map[string]string{
		"sleep1": "signaled(15)",
		"sleep2": "signaled(9)",
		"sleep3": "running",
		"true":   "exited(0)",
		"false":  "exited(1)",
	})
}

func TestBadStatusRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	tests := [][][]string{
		// Test 1: Request Extra cruft after STATUS
		[][]string{
			[]string{"STATUS", "EXTRA"},
		},

		// Test 2: Request is too long..
		[][]string{
			[]string{"STATUS"},
			[]string{},
		},
	}
	BadResultsCheck(t, tests)
}
