// Copyright 2013-2015 Apcera Inc. All rights reserved.

// +build linux,cgo

package stage3_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/apcera/util/testtool"
)

func TestExecRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// Start the initd process.
	cgroup, socket, _, pid := StartInitd(t)

	// Make a request against the initd server.
	dir := TempDir(t)
	stdout := path.Join(dir, "stdout")
	stderr := path.Join(dir, "stderr")
	cmd := []string{"/bin/sleep", "60"}
	request := [][]string{
		[]string{"EXEC"},
		cmd,
		[]string{"KTEST=VTEST"},
		[]string{stdout, stderr},
	}
	reply, err := MakeRequest(socket, request, 10*time.Second)
	TestExpectSuccess(t, err)
	TestEqual(t, reply, "REQUEST OK\n")
	sleepPid, tasks := waitTask(t, cgroup, cmd, 5*time.Second)

	// Check the command line of both the tasks and see if the proper process
	// tree exists for what we would expect:
	// 		(parent) /sbin/sleep
	//		    |--- (child) os.Argv[0]
	// Ensure correct parent properties
	cmdline, env, ppid, children := taskInfo(t, sleepPid)
	TestEqual(t, cmdline, []string{"/bin/sleep", "60"})
	TestEqual(t, env, []string{"KTEST=VTEST"})
	TestEqual(t, ppid, pid)
	TestEqual(t, children, []int{tasks[2]})

	// Ensure correct child properties
	_, t1env, t1ppid, t1children := taskInfo(t, tasks[2])
	TestEqual(t, t1env, []string{"INITD_DEBUG=1", "INITD_INTERCEPT=1", "INITD_SOCKET=" + socket})
	TestEqual(t, t1ppid, tasks[1])
	TestEqual(t, t1children, []int{})

	// Check the three normal file descriptors, 0 -> /dev/null, 1 -> stdout,
	// 2 -> stderr (stdout/stderr were allocated earlier.)
	stdinLink, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/0", sleepPid))
	TestExpectSuccess(t, err)
	TestEqual(t, stdinLink, "/dev/null")
	stdoutLink, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/1", sleepPid))
	TestExpectSuccess(t, err)
	TestEqual(t, stdoutLink, stdout)
	stderrLink, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/2", sleepPid))
	TestExpectSuccess(t, err)
	TestEqual(t, stderrLink, stderr)

	// Check that the file descriptors on the sleep binary are all closed and
	// nothing was shared that was not supposed to be. Walk the list and check
	// that each and every value is normal.
	dirs, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/fd", sleepPid))
	TestExpectSuccess(t, err)
	TestEqual(t, len(dirs), 3)
}

func TestBadExecRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	tests := [][][]string{
		// Test 1: Request is too short.
		[][]string{
			[]string{"EXEC"},
			[]string{"COMMAND"},
			[]string{"ENVKEY=ENVVALUE"},
		},

		// Test 2: Request is too long..
		[][]string{
			[]string{"EXEC"},
			[]string{"COMMAND"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR"},
			[]string{},
		},

		// Test 3: No command defined.
		[][]string{
			[]string{"EXEC"},
			[]string{},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR"},
		},

		// Test 4: No chroot directory defined.
		[][]string{
			[]string{"EXEC"},
			[]string{},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR"},
		},

		// Test 5: Extra cruft after EXEC
		[][]string{
			[]string{"EXEC", "EXTRA"},
			[]string{"COMMAND"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR"},
		},

		// Test 6: Extra cruft after STDERR.
		[][]string{
			[]string{"EXEC"},
			[]string{"COMMAND"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR", "EXTRA"},
		},

		// Test 7: Extra cruft after CHROOT.
		[][]string{
			[]string{"EXEC"},
			[]string{"COMMAND"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR", "EXTRA"},
		},
	}
	BadResultsCheck(t, tests)
}
