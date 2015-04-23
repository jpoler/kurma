// Copyright 2013-2015 Apcera Inc. All rights reserved.

// +build linux,cgo

package stage3

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/apcera/util/testtool"
)

func TestUnnamedStartRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// Start the initd process.
	cgroup, socket, _, pid := StartInitd(t)

	// SLEEP PROCESS 1

	// Make a request against the initd server.
	dir := TempDir(t)
	stdout := path.Join(dir, "stdout")
	stderr := path.Join(dir, "stderr")
	sleep1 := []string{"/bin/sleep", "60"}
	request := [][]string{
		[]string{"START"},
		sleep1,
		[]string{""},
		[]string{"KTEST=VTEST"},
		[]string{stdout, stderr},
		[]string{"99", "99"},
	}
	reply, err := MakeRequest(socket, request, 10*time.Second)
	TestExpectSuccess(t, err)
	TestEqual(t, reply, "REQUEST OK\n")
	s1pid, tasks := waitTask(t, cgroup, sleep1, 5*time.Second)
	TestEqual(t, len(tasks), 2)

	// Get the task information for the sleep process.
	s1cmd, s1env, s1ppid, s1children := taskInfo(t, s1pid)
	TestEqual(t, s1cmd, []string{"/bin/sleep", "60"})
	TestEqual(t, s1env, []string{"KTEST=VTEST"})
	TestEqual(t, s1ppid, pid)
	TestEqual(t, s1children, []int{})

	// Check the three normal file descriptors, 0 -> /dev/null, 1 -> stdout,
	// 2 -> stderr (stdout/stderr were allocated earlier.)
	stdinLink, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/0", s1pid))
	TestExpectSuccess(t, err)
	TestEqual(t, stdinLink, "/dev/null")
	stdoutLink, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/1", s1pid))
	TestExpectSuccess(t, err)
	TestEqual(t, stdoutLink, stdout)
	stderrLink, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/2", s1pid))
	TestExpectSuccess(t, err)
	TestEqual(t, stderrLink, stderr)

	// Check that the file descriptors on the sleep binary are all closed and
	// nothing was shared that was not supposed to be. Walk the list and check
	// that each and every value is normal.
	dirs, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/fd", s1pid))
	TestExpectSuccess(t, err)
	TestEqual(t, len(dirs), 3)

	// FIXME: Check the UID/GID

	// SLEEP PROCESS 2

	// Make a request against the initd server.
	dir = TempDir(t)
	stdout = path.Join(dir, "stdout")
	stderr = path.Join(dir, "stderr")
	sleep2 := []string{"/bin/sleep", "61"}
	request = [][]string{
		[]string{"START"},
		sleep2,
		[]string{""},
		[]string{"KTEST2=VTEST2", "KTEST3=VTEST3"},
		[]string{stdout, stderr},
		[]string{"99", "99"},
	}
	reply, err = MakeRequest(socket, request, 10*time.Second)
	TestExpectSuccess(t, err)
	TestEqual(t, reply, "REQUEST OK\n")
	s2pid, tasks := waitTask(t, cgroup, sleep2, 5*time.Second)

	// Get the task information for the 2nd sleep process.
	s2cmd, s2env, s2ppid, s2children := taskInfo(t, s2pid)
	TestEqual(t, s2cmd, []string{"/bin/sleep", "61"})
	TestEqual(t, s2env, []string{"KTEST2=VTEST2", "KTEST3=VTEST3"})
	TestEqual(t, s2ppid, pid)
	TestEqual(t, s2children, []int{})

	// Check the three normal file descriptors, 0 -> /dev/null, 1 -> stdout,
	// 2 -> stderr (stdout/stderr were allocated earlier.)
	stdinLink, err = os.Readlink(fmt.Sprintf("/proc/%d/fd/0", s2pid))
	TestExpectSuccess(t, err)
	TestEqual(t, stdinLink, "/dev/null")
	stdoutLink, err = os.Readlink(fmt.Sprintf("/proc/%d/fd/1", s2pid))
	TestExpectSuccess(t, err)
	TestEqual(t, stdoutLink, stdout)
	stderrLink, err = os.Readlink(fmt.Sprintf("/proc/%d/fd/2", s2pid))
	TestExpectSuccess(t, err)
	TestEqual(t, stderrLink, stderr)

	// Check that the file descriptors on the sleep binary are all closed and
	// nothing was shared that was not supposed to be. Walk the list and check
	// that each and every value is normal.
	dirs, err = ioutil.ReadDir(fmt.Sprintf("/proc/%d/fd", s2pid))
	TestExpectSuccess(t, err)
	TestEqual(t, len(dirs), 3)

	// FIXME: Check the UID/GID
}

func TestNamedStartRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// Start the initd process.
	cgroup, socket, _, pid := StartInitd(t)

	// SLEEP PROCESS 1

	// Make a request against the initd server.
	dir := TempDir(t)
	stdout := path.Join(dir, "stdout")
	stderr := path.Join(dir, "stderr")
	sleep1 := []string{"/bin/sleep", "60"}
	request := [][]string{
		[]string{"START", "SLEEP1"},
		sleep1,
		[]string{""},
		[]string{"KTEST=VTEST"},
		[]string{stdout, stderr},
		[]string{"99", "99"},
	}
	reply, err := MakeRequest(socket, request, 10*time.Second)
	TestExpectSuccess(t, err)
	TestEqual(t, reply, "REQUEST OK\n")
	s1pid, tasks := waitTask(t, cgroup, sleep1, 5*time.Second)
	TestEqual(t, len(tasks), 2)

	// Get the task information for the sleep process.
	s1cmd, s1env, s1ppid, s1children := taskInfo(t, s1pid)
	TestEqual(t, s1cmd, []string{"/bin/sleep", "60"})
	TestEqual(t, s1env, []string{"KTEST=VTEST"})
	TestEqual(t, s1ppid, pid)
	TestEqual(t, s1children, []int{})

	// Check the three normal file descriptors, 0 -> /dev/null, 1 -> stdout,
	// 2 -> stderr (stdout/stderr were allocated earlier.)
	stdinLink, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/0", s1pid))
	TestExpectSuccess(t, err)
	TestEqual(t, stdinLink, "/dev/null")
	stdoutLink, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/1", s1pid))
	TestExpectSuccess(t, err)
	TestEqual(t, stdoutLink, stdout)
	stderrLink, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/2", s1pid))
	TestExpectSuccess(t, err)
	TestEqual(t, stderrLink, stderr)

	// Check that the file descriptors on the sleep binary are all closed and
	// nothing was shared that was not supposed to be. Walk the list and check
	// that each and every value is normal.
	dirs, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/fd", s1pid))
	TestExpectSuccess(t, err)
	TestEqual(t, len(dirs), 3, fmt.Sprintf("Found fds: %#v", dirs))

	// SLEEP PROCESS 2

	// Make a request against the initd server.
	dir = TempDir(t)
	stdout = path.Join(dir, "stdout")
	stderr = path.Join(dir, "stderr")
	sleep2 := []string{"/bin/sleep", "61"}
	request = [][]string{
		[]string{"START", "START2"},
		sleep2,
		[]string{""},
		[]string{"KTEST2=VTEST2", "KTEST3=VTEST3"},
		[]string{stdout, stderr},
		[]string{"99", "99"},
	}
	reply, err = MakeRequest(socket, request, 10*time.Second)
	TestExpectSuccess(t, err)
	TestEqual(t, reply, "REQUEST OK\n")
	s2pid, tasks := waitTask(t, cgroup, sleep2, 5*time.Second)

	// Get the task information for the sleep process.
	s2cmd, s2env, s2ppid, s2children := taskInfo(t, s2pid)
	TestEqual(t, s2cmd, sleep2)
	TestEqual(t, s2env, []string{"KTEST2=VTEST2", "KTEST3=VTEST3"})
	TestEqual(t, s2ppid, pid)
	TestEqual(t, s2children, []int{})

	// Check the three normal file descriptors, 0 -> /dev/null, 1 -> stdout,
	// 2 -> stderr (stdout/stderr were allocated earlier.)
	stdinLink, err = os.Readlink(fmt.Sprintf("/proc/%d/fd/0", s2pid))
	TestExpectSuccess(t, err)
	TestEqual(t, stdinLink, "/dev/null")
	stdoutLink, err = os.Readlink(fmt.Sprintf("/proc/%d/fd/1", s2pid))
	TestExpectSuccess(t, err)
	TestEqual(t, stdoutLink, stdout)
	stderrLink, err = os.Readlink(fmt.Sprintf("/proc/%d/fd/2", s2pid))
	TestExpectSuccess(t, err)
	TestEqual(t, stderrLink, stderr)

	// Check that the file descriptors on the sleep binary are all closed and
	// nothing was shared that was not supposed to be. Walk the list and check
	// that each and every value is normal.
	dirs, err = ioutil.ReadDir(fmt.Sprintf("/proc/%d/fd", s2pid))
	TestExpectSuccess(t, err)
	TestEqual(t, len(dirs), 3, fmt.Sprintf("Found fds: %#v", dirs))
}

func TestBadStartRequest(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	tests := [][][]string{
		// Test 1: Request is too short.
		[][]string{
			[]string{"START"},
			[]string{"COMMAND"},
			[]string{"DIR"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR"},
		},

		// Test 2: Request is too long..
		[][]string{
			[]string{"START"},
			[]string{"COMMAND"},
			[]string{"DIR"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR"},
			[]string{"UID", "GID"},
			[]string{},
		},

		// Test 3: Extra cruft after NAME
		[][]string{
			[]string{"START", "NAME", "EXTRA"},
			[]string{"COMMAND"},
			[]string{"DIR"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR"},
			[]string{"UID", "GID"},
		},

		// Test 4: Extra cruft after STDERR
		[][]string{
			[]string{"START"},
			[]string{"COMMAND"},
			[]string{"DIR"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR", "EXTRA"},
			[]string{"UID", "GID"},
		},

		// Test 5: Extra cruft after GID
		[][]string{
			[]string{"START"},
			[]string{"COMMAND"},
			[]string{"DIR"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR"},
			[]string{"UID", "GID", "EXTRA"},
		},

		// Test 6: Missing COMMAND
		[][]string{
			[]string{"START"},
			[]string{},
			[]string{"DIR"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR"},
			[]string{"UID", "GID"},
		},

		// Test 7: Too many args with WORKING DIR
		[][]string{
			[]string{"START"},
			[]string{"COMMAND"},
			[]string{"DIR", "DOH"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR"},
			[]string{"UID", "GID"},
		},

		// Test 8: Missing STDERR
		[][]string{
			[]string{"START"},
			[]string{"COMMAND"},
			[]string{"DIR"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT"},
			[]string{"UID", "GID"},
		},

		// Test 9: Missing STDOUT
		[][]string{
			[]string{"START"},
			[]string{"COMMAND"},
			[]string{"DIR"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{},
			[]string{"UID", "GID"},
		},

		// Test 10: Missing GID
		[][]string{
			[]string{"START"},
			[]string{"COMMAND"},
			[]string{"DIR"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR"},
			[]string{"UID"},
		},

		// Test 11: Missing UID
		[][]string{
			[]string{"START"},
			[]string{"COMMAND"},
			[]string{"DIR"},
			[]string{"ENVKEY=ENVVALUE"},
			[]string{"STDOUT", "STDERR"},
			[]string{},
		},
	}
	BadResultsCheck(t, tests)
}
