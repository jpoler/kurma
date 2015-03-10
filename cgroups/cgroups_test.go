// Copyright 2013 Apcera Inc. All rights reserved.

// +build linux,cgo

package cgroups

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/apcera/util/uuid"

	. "github.com/apcera/util/testtool"
)

const EXECUTION_TOKEN = "9s8d0fsdkmf.2/,3098wdf"

// We need to be able to intercept execution so we can inject signal handling in
// a "child." To do this we ensure that we can fork/exec the built test
// executable and have it do the childs work for us.
func init() {
	if len(os.Args) != 2 {
		return
	} else if os.Args[1] != EXECUTION_TOKEN {
		return
	}

	usr1 := make(chan os.Signal, 100)
	usr2 := make(chan os.Signal, 100)
	signal.Notify(usr1, syscall.SIGUSR1)
	signal.Notify(usr2, syscall.SIGUSR2)

	// Send a ready message to the user.
	fmt.Fprintf(os.Stderr, "READY\n")
	os.Stderr.Close()

	for {
		select {
		case <-time.NewTimer(time.Second * 3).C:
			fmt.Println("TIMEOUT")
			os.Exit(1)
		case <-usr1:
			fmt.Println("SIGUSR1")
			os.Exit(0)
		case <-usr2:
			fmt.Println("SIGUSR2")
			os.Exit(0)
		}
	}
}

func StartSignalWatcher(t *testing.T, cgroupdir string) (*exec.Cmd, *os.File) {
	//
	cmd := exec.Command(os.Args[0], EXECUTION_TOKEN)
	output, stdout, err := os.Pipe()
	if err != nil {
		t.Fatalf("Error making pipe: %s", err)
	}
	defer stdout.Close()
	check, stderr, err := os.Pipe()
	if err != nil {
		t.Fatalf("Error making pipe: %s", err)
	}
	defer stderr.Close()

	cmd.Stdin = nil
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("Error starting signal process: %s", err)
	}

	// Add this task to the given cgroup.
	if cgroupdir != "" {
		// Add the task to the cgroup
		tasksfile := path.Join(cgroupdir, "tasks")
		pidstr := []byte(fmt.Sprintf("%d\n", cmd.Process.Pid))
		if err := ioutil.WriteFile(tasksfile, pidstr, 0644); err != nil {
			t.Fatalf("Error adding process to cgroup: %s", err)
		}
	}

	rchan := make(chan string)
	go func() {
		line, err := bufio.NewReader(check).ReadString('\n')
		if err != nil {
			rchan <- fmt.Sprintf("Error waiting for process start: %s", err)
		} else if line != "READY\n" {
			rchan <- fmt.Sprintf("Wrong string read: %s", line)
		} else {
			rchan <- ""
		}
	}()
	select {
	case msg := <-rchan:
		if msg != "" {
			t.Fatalf(msg)
		}
	case <-time.NewTimer(time.Second).C:
		t.Fatalf("timeout starting command")
	}
	return cmd, output
}

// ----------------
// Helper Functions
// ----------------

func ShotgunZombieProcs(t *testing.T) {
	for {
		pid, err := syscall.Wait4(-1, nil, syscall.WNOHANG, nil)
		if err != nil {
			Fatalf(t, "Unexpected Wait4 error: %s", err)
		} else if pid == 0 {
			return
		}
	}
}

func MakeUniqueCgroup(t *testing.T) (string, *Cgroup) {
	uniquename := fmt.Sprintf("unittest-%s", uuid.Variant4().String())
	cgroup, err := New(uniquename)
	if err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}
	return uniquename, cgroup
}

// Wrapper for cgroups cleanup.
func CleanupCgroup(t *testing.T, c *Cgroup) {
	if err := c.Destroy(); err != nil {
		Fatalf(t, "Error removing the cgroups we successfully created.")
	}
}

// ----------------------
// Cgroups Function Tests
// ----------------------

func TestNew(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// ------------------
	// Failure Conditions
	// ------------------

	// Test 1: Tasks is not autopopulated
	func() {
		defer func(c string) { cgroupsDir = c }(cgroupsDir)
		cgroupsDir = TempDir(t)
		if _, err := New("test"); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 2: Cgroup already exists.
	func() {
		defer func(c string) { cgroupsDir = c }(cgroupsDir)
		cgroupsDir = TempDir(t)
		fn := path.Join(cgroupsDir, defaultCgroups[0], "test")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		if _, err := New("test"); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 3: Cgroup directory is actually a file.
	func() {
		defer func(c string) { cgroupsDir = c }(cgroupsDir)
		cgroupsDir = TempDir(t)
		fn := path.Join(cgroupsDir, defaultCgroups[0])
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		fn = path.Join(cgroupsDir, defaultCgroups[0], "test")
		if err := ioutil.WriteFile(fn, []byte{}, 0644); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		if _, err := New("test"); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 4: Make the directory creation fail.
	func() {
		defer func(c string) { cgroupsDir = c }(cgroupsDir)
		defer func() { osMkdir = os.Mkdir }()
		osMkdir = func(name string, mode os.FileMode) error {
			return fmt.Errorf("expected error from Mkdir()")
		}

		cgroupsDir = TempDir(t)
		fn := path.Join(cgroupsDir, defaultCgroups[0])
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		if _, err := New("test"); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 5: Make the directory stat fail.
	func() {
		defer func(c string) { cgroupsDir = c }(cgroupsDir)
		defer func() { osLstat = os.Lstat }()
		osLstat = func(name string) (os.FileInfo, error) {
			return nil, fmt.Errorf("expected error from os.Lstat()")
		}

		cgroupsDir = TempDir(t)
		fn := path.Join(cgroupsDir, defaultCgroups[0])
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		if _, err := New("test"); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 6: Existing tasks in the cgroup.
	func() {
		// A unique name to use for cgroup names.
		uniquename := fmt.Sprintf("unittest-%s", uuid.Variant4().String())

		cgroup := path.Join(cgroupsDir, "memory", uniquename)

		// Ensure that we remove this cgroups when the test finishes.
		defer func() {
			// We loop to see if we can remove this for 5 seconds.
			end := time.Now().Add(time.Second * 5)
			for time.Now().Before(end) {
				if err := os.Remove(cgroup); err == nil {
					return
				}
				time.Sleep(time.Millisecond * 100)
			}
			Fatalf(t, "Error removing cgroups directory.")
		}()

		if err := os.Mkdir(cgroup, 0755); err != nil {
			if !os.IsExist(err) {
				Fatalf(t, "Unexpected error: %s", err)
			}
		}

		// Put a signal watcher in the cgroup.
		cmd, fd := StartSignalWatcher(t, cgroup)

		// Kill the process we created so the test cleans up properly.
		defer func() {
			// Kill the child process.
			if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
				t.Errorf("Error killing the child: %s", err)
			}
			// Read from the stdout for the process until it dies.
			if _, err := ioutil.ReadAll(fd); err != nil {
				t.Errorf("Error waiting for child to die: %s", err)
			}
		}()

		// Now attempt to create the cgroup which should fail because it already has
		// tasks in it.
		if _, err := New(uniquename); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// -------
	// Success
	// -------

	// A unique name to use for cgroup names.
	_, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)
}

func TestCgroup_AddTask(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// A valid cgroup that we will use in the tests.
	uniquename, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)

	// ------------------
	// Failure Conditions
	// ------------------

	// test 1: Cgroup directories do not exist at all.
	func() {
		cgroup := Cgroup{name: "doesnotexist"}
		if err := cgroup.AddTask(os.Getpid()); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 2: Can not add the pid to the cgroup. (PID doesn't exist)
	func() {
		// Find a pid that doesn't exist.
		var pid int
		for pid = 1000; true; pid += 1 {
			if _, err := os.Lstat(fmt.Sprintf("/proc/%d", pid)); err != nil {
				if os.IsNotExist(err) {
					break
				}
			}
		}

		// Now add that pid to the cgroup.
		if err := cgroup.AddTask(pid); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// -------
	// Success
	// -------

	cmd, fd := StartSignalWatcher(t, "")
	if err := cgroup.AddTask(cmd.Process.Pid); err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}
	defer fd.Close()

	// Verify that the cgroup file in proc reflects this.
	pid := cmd.Process.Pid
	fn := fmt.Sprintf("/proc/%d/task/%d/cgroup", pid, pid)
	if data, err := ioutil.ReadFile(fn); err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	} else {
		// Make a map of cgroup type to cgroup
		groups := make(map[string]string)
		for _, line := range strings.Split(string(data), "\n") {
			if line == "" {
				continue
			}
			parts := strings.Split(line, ":")
			if len(parts) != 3 {
				continue
			}
			groups[parts[1]] = parts[2]
		}

		for _, ctype := range defaultCgroups {
			if groups[ctype] != "/"+uniquename {
				Fatalf(t, "Process %d not in the right cgroup", pid)
			}
		}
	}
}

func TestCgroup_CPUUsed(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// A valid cgroup that we will use in the tests.
	_, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)

	// ------------------
	// Failure Conditions
	// ------------------

	// Test 1: File not found.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: ""}

		if _, err := cgroup.CPUUsed(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 3: Too many columns.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: "tmp"}

		// Make the directory.
		fn := path.Join(cgroupsDir, "cpuacct", "tmp")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		fn = path.Join(fn, "cpuacct.usage")
		contents := []byte("1 1\n")
		if err := ioutil.WriteFile(fn, contents, 0644); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		if _, err := cgroup.CPUUsed(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 4: Not an integer.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: "tmp"}

		// Make the directory.
		fn := path.Join(cgroupsDir, "cpuacct", "tmp")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		fn = path.Join(fn, "cpuacct.usage")
		contents := []byte("xyz\n")
		if err := ioutil.WriteFile(fn, contents, 0644); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		if _, err := cgroup.CPUUsed(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// -------
	// Success
	// -------

	// Make a cpu consuming task. This will burn CPU until the unit test finishes
	// or it is killed.
	cmd := exec.Command(
		"/usr/bin/nice", "/bin/bash", "-c",
		fmt.Sprintf("while test -d /proc/%d ; do true ; done", os.Getpid()))
	if err := cmd.Start(); err != nil {
		Fatalf(t, "Error starting command: %s", err)
	}
	if err := cgroup.AddTask(cmd.Process.Pid); err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	orig, err := cgroup.CPUUsed()
	if err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	// Loop to ensure that we see CPU utilization happening.
	end := time.Now().Add(time.Second * 5)
	for {
		if time.Now().After(end) {
			Fatalf(t, "Timeout waiting for CPU use to be noticed.")
		}
		current, err := cgroup.CPUUsed()
		if err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		} else if current > orig {
			break
		}
	}
}

func TestCgroup_Children(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// ------------------
	// Failure Conditions
	// ------------------

	// Test 1: Cgroups directory isn't a directory.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: "tmp"}

		fn := path.Join(cgroupsDir, defaultCgroups[0])
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		fn = path.Join(fn, "tmp")
		if err := ioutil.WriteFile(fn, []byte{}, 0644); err != nil {
			t.Fatalf("Error adding process to cgroup: %s", err)
		}

		if clist, err := cgroup.Children(); err == nil {
			Fatalf(t, "Expected error not returned.")
		} else if len(clist) != 0 {
			Fatalf(t, "Unexpected results: %#v", clist)
		}
	}()

	// ------------------
	// Success Conditions
	// ------------------

	// Test 1: Cgroup directory doesn't even exist.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: "tmp"}

		if clist, err := cgroup.Children(); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		} else if len(clist) != 0 {
			Fatalf(t, "Unexpected results: %#v", clist)
		}
	}()

	// Test 2: Cgroup has children.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: "tmp"}

		// Three unique child cgroups spread across the names in
		// defaultCgroups.
		fn := path.Join(cgroupsDir, defaultCgroups[0], "tmp", "child1")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		fn = path.Join(cgroupsDir, defaultCgroups[1], "tmp", "child2")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		fn = path.Join(cgroupsDir, defaultCgroups[2], "tmp", "child3")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		// A duplicate cgroup name to ensure that deduping works.
		fn = path.Join(cgroupsDir, defaultCgroups[1], "tmp", "child1")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		if clist, err := cgroup.Children(); err != nil {
			Fatalf(t, "Unexpected error not returned: %s", err)
		} else if len(clist) != 3 {
			Fatalf(t, "Unexpected results: %#v", clist)
		} else {
			found := make([]string, len(clist))
			for i, c := range clist {
				found[i] = c.name
			}
			expected := []string{"tmp/child1", "tmp/child2", "tmp/child3"}
			sort.Strings(expected)
			sort.Strings(found)
			if !reflect.DeepEqual(found, expected) {
				Fatalf(t, "Unexpected children: %s", found)
			}
		}
	}()
}

func TestCgroup_Destroy(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// ------------------
	// Failure Conditions
	// ------------------

	// Test 1: Cgroups directory isn't a directory.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: "tmp"}

		fn := path.Join(cgroupsDir, defaultCgroups[0])
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		fn = path.Join(fn, "tmp")
		if err := ioutil.WriteFile(fn, []byte{}, 0644); err != nil {
			t.Fatalf("Error adding process to cgroup: %s", err)
		}

		if err := cgroup.Destroy(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 2: Cgroups child directory directory isn't a directory.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: ""}

		fn := path.Join(cgroupsDir, defaultCgroups[0], "child1", "child2")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		fn = path.Join(fn, "tmp")
		if err := ioutil.WriteFile(fn, []byte{}, 0644); err != nil {
			t.Fatalf("Error adding process to cgroup: %s", err)
		}

		if err := cgroup.Destroy(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 3: Error from SignalAll
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: ""}

		fn := path.Join(cgroupsDir, defaultCgroups[0])
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		fn = path.Join(fn, "tasks")
		contents := []byte("not_an_int\n")
		if err := ioutil.WriteFile(fn, contents, 0644); err != nil {
			t.Fatalf("Error adding process to cgroup: %s", err)
		}

		if err := cgroup.Destroy(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 4: Error from Shutdown()
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: "test"}

		fn := path.Join(cgroupsDir, defaultCgroups[0], "test")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		tasksfile := path.Join(fn, "tasks")
		contents := []byte("")
		if err := ioutil.WriteFile(tasksfile, contents, 0644); err != nil {
			t.Fatalf("Error adding process to cgroup: %s", err)
		}
		otherfile := path.Join(fn, "other")
		contents = []byte("")
		if err := ioutil.WriteFile(otherfile, contents, 0644); err != nil {
			t.Fatalf("Error adding process to cgroup: %s", err)
		}

		if err := cgroup.Destroy(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// ------------------
	// Success Conditions
	// ------------------

	_, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)

	cgroupChild1, err := cgroup.New("child1")
	if err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	// Start processes to put in each cgroup.
	pids := make([]int, 0)
	for i := 0; i < 5; i++ {
		cmd, fd := StartSignalWatcher(t, "")
		if err := cgroup.AddTask(cmd.Process.Pid); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		defer fd.Close()
		pids = append(pids, cmd.Process.Pid)
		cmd, fd = StartSignalWatcher(t, "")
		if err := cgroupChild1.AddTask(cmd.Process.Pid); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		pids = append(pids, cmd.Process.Pid)
		defer fd.Close()
	}

	// Now attempt to destroy the cgroup.
	if err := cgroup.Destroy(); err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	} else {
		cgroup = nil
	}

	// Give the kernel a moment to actually do what it needs to do.
	time.Sleep(time.Millisecond * 100)

	// Clear out the zombies to make this next check cleaner.
	ShotgunZombieProcs(t)

	// Check that all the pids no longer exist.
	for _, pid := range pids {
		if _, err := os.Lstat(fmt.Sprintf("/proc/%d", pid)); err != nil {
			if !os.IsNotExist(err) {
				Fatalf(t, "Unexpected error: %s", err)
			}
		} else {
			Fatalf(t, "Process failed to die: %d", pid)
		}
	}
}

func TestCgroup_Destroyed(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// ------------------
	// Failure Conditions
	// ------------------

	// Test 1: Make the directory stat fail.
	func() {
		defer func() { osLstat = os.Lstat }()
		osLstat = func(name string) (os.FileInfo, error) {
			return nil, fmt.Errorf("expected error from os.Lstat()")
		}

		cgroup := Cgroup{name: "test"}
		if _, err := cgroup.Destroyed(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// ------------------
	// Success Conditions
	// ------------------

	// Test 1: At least one directory exists.
	func() {
		defer func(c string) { cgroupsDir = c }(cgroupsDir)
		cgroupsDir = TempDir(t)

		cgroup := Cgroup{name: "test"}
		fn := path.Join(cgroupsDir, defaultCgroups[0], "test")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		if res, err := cgroup.Destroyed(); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		} else if res == true {
			Fatalf(t, "Invalid response from Destroyed()")
		}
	}()

	// Test 2: The last directory exists.
	func() {
		defer func(c string) { cgroupsDir = c }(cgroupsDir)
		cgroupsDir = TempDir(t)

		cgroup := Cgroup{name: "test"}
		ctype := defaultCgroups[len(defaultCgroups)-1]
		fn := path.Join(cgroupsDir, ctype, "test")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		if res, err := cgroup.Destroyed(); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		} else if res == true {
			Fatalf(t, "Invalid response from Destroyed()")
		}
	}()

	// Test 3: No directories exist.
	func() {
		defer func(c string) { cgroupsDir = c }(cgroupsDir)
		cgroupsDir = TempDir(t)

		cgroup := Cgroup{name: "test"}
		fn := path.Join(cgroupsDir, defaultCgroups[0])
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		if res, err := cgroup.Destroyed(); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		} else if res == false {
			Fatalf(t, "Invalid response from Destroyed()")
		}
	}()
}

func TestCgroup_LimitCPU(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// ------------------
	// Failure Conditions
	// ------------------

	// Test 1: cpu.cfs_quota_us is unwritable.
	func() {
		defer func(c string) { cgroupsDir = c }(cgroupsDir)
		cgroupsDir = TempDir(t)

		cgroup := Cgroup{name: "test"}
		fn := filepath.Join(cgroupsDir, "cpu", "test", cpuQuota)
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		if err := cgroup.LimitCPU(1000); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 2: cpu.cfs_period_us is unwritable.
	func() {
		defer func(c string) { cgroupsDir = c }(cgroupsDir)
		cgroupsDir = TempDir(t)

		cgroup := Cgroup{name: "test"}
		fn := filepath.Join(cgroupsDir, "cpu", "test", cpuPeriod)
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		if err := cgroup.LimitCPU(1000); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// ------------------
	// Success Conditions
	// ------------------

	uniquename, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)

	limit := int64(100)

	// Attempt to set the actual CPU limit.
	if err := cgroup.LimitCPU(limit); err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	// Read the values back and make sure that they make sense.
	fn := filepath.Join(cgroupsDir, "cpu", uniquename, cpuQuota)
	quotaBytes, err := ioutil.ReadFile(fn)
	if err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}
	fn = filepath.Join(cgroupsDir, "cpu", uniquename, cpuPeriod)
	periodBytes, err := ioutil.ReadFile(fn)
	if err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	fn = filepath.Join(cgroupsDir, "cpu", cpuPeriod)
	expectedPeriod, err := ioutil.ReadFile(fn)
	TestExpectSuccess(t, err)

	TestEqual(t, quotaBytes, []byte("10000\n"))
	TestEqual(t, periodBytes, expectedPeriod)
}

func TestCgroup_LimitMemory(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// ------------------
	// Failure Conditions
	// ------------------

	// Test 1: memory.limit_in_bytes is unwritable.
	func() {
		defer func(c string) { cgroupsDir = c }(cgroupsDir)
		cgroupsDir = TempDir(t)

		cgroup := Cgroup{name: "test"}
		fn := path.Join(cgroupsDir, "memory", "test", memLimit)
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		if err := cgroup.LimitMemory(1000); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// ------------------
	// Success Conditions
	// ------------------

	uniquename, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)

	limit := int64(1024 * 1024)

	// Attempt to set the actual Memory limit.
	if err := cgroup.LimitMemory(limit); err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	// Read the values back and make sure that they make sense.
	fn := path.Join(cgroupsDir, "memory", uniquename, memLimit)
	limitBytes, err := ioutil.ReadFile(fn)
	if err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}
	limitStr := strings.Trim(string(limitBytes), "\n")
	actuallimit, err := strconv.ParseInt(limitStr, 10, 63)
	if err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	if actuallimit != limit {
		Fatalf(t, "Read value was not expected: %d", actuallimit)
	}
}

func TestCgroup_MemoryUsed(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// A valid cgroup that we will use in the tests.
	_, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)

	// ------------------
	// Failure Conditions
	// ------------------

	// Test 1: File not found.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: "foo"}

		if _, err := cgroup.MemoryUsed(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 2: Not an integer.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: "tmp"}

		// Make the directory.
		fn := path.Join(cgroupsDir, "memory", "tmp")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		fn = path.Join(fn, memUsage)
		contents := []byte("xyz\n")
		if err := ioutil.WriteFile(fn, contents, 0644); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		if _, err := cgroup.MemoryUsed(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// -------
	// Success
	// -------

	// Make a memory consuming task. This will incrementally consume more
	// and more memory until it is at ~1MB.
	cmd := exec.Command("/usr/bin/sort")
	writer, err := cmd.StdinPipe()
	if err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}
	if err := cmd.Start(); err != nil {
		Fatalf(t, "Error starting command: %s", err)
	}
	if err := cgroup.AddTask(cmd.Process.Pid); err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	// Check the initial memory use.
	orig, err := cgroup.MemoryUsed()
	if err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	// Write a bunch of data to the writer socket so the child will
	// consume some memory.
	totalsize := int64(0)
	for i := 0; i < 100000; i++ {
		size, err := io.WriteString(writer, fmt.Sprintf("line-%d", i))
		if err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		totalsize += int64(size)
	}

	// Get the memory again now that we shoved data in.
	ending, err := cgroup.MemoryUsed()
	if err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	// Verify that more memory was used at the end of the run.
	if ending <= orig {
		Fatalf(
			t, "Ending memory (%d) was not greater than orig (%d)",
			ending, orig)
	}
}

func TestCgroup_New(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// A valid cgroup that we will use in the tests.
	_, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)

	// Make a child. This is mostly tested in New() so not much do to here.
	child, err := cgroup.New("child")
	if err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	// Verify that the name is set properly.
	if child.name != path.Join(cgroup.name, "child") {
		Fatalf(t, "Bad child name set: %s", cgroup.name)
	}
}

func TestCgroup_Name(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// A valid cgroup that we will use in the tests.
	uniquename, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)

	// Verify that the name is set properly.
	if cgroup.Name() != uniquename {
		Fatalf(t, "Name not set properly: %s", cgroup.Name())
	}
}

func TestCgroup_Tasks(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// ------------------
	// Failure Conditions
	// ------------------

	// Test 1: Error from ioutil.ReadFile
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)

		// Reset the ioutilReadFile function.
		defer func() { ioutilReadFile = ioutil.ReadFile }()
		ioutilReadFile = func(name string) ([]byte, error) {
			return nil, fmt.Errorf("expected error from ioutil.ReadFile()")
		}

		cgroup := Cgroup{name: "foo"}
		if _, err := cgroup.Tasks(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 2: Too many columns.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: "tmp"}

		// Make the directory.
		fn := path.Join(cgroupsDir, "memory", "tmp")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		fn = path.Join(fn, "tasks")
		contents := []byte("1 1\n")
		if err := ioutil.WriteFile(fn, contents, 0644); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		if _, err := cgroup.Tasks(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 3: Task line is not a number.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)
		cgroup := Cgroup{name: "tmp"}

		// Make the directory.
		fn := path.Join(cgroupsDir, "memory", "tmp")
		if err := os.MkdirAll(fn, 0755); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		fn = path.Join(fn, "tasks")
		contents := []byte("not_an_init\n")
		if err := ioutil.WriteFile(fn, contents, 0644); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		if _, err := cgroup.Tasks(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// -------
	// Success
	// -------

	// A valid cgroup that we will use in the tests.
	_, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)

	// Startup 10 sleep commands into the container.
	pids := make([]int, 10)
	for i := 0; i < 10; i++ {
		cmd := exec.Command("/bin/sleep", "60")
		if err := cmd.Start(); err != nil {
			Fatalf(t, "Error starting command: %s", err)
		}
		if err := cgroup.AddTask(cmd.Process.Pid); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}
		pids[i] = cmd.Process.Pid
	}

	// Check to see that all the pids show up in the tasks list. We should not
	// have threads in the sleep call so we shouldn't have to deal with the
	// difference between tasks and pids.. etc.
	tasks, err := cgroup.Tasks()
	if err != nil {
		Fatalf(t, "Unexpected error in Tasks(): %s", err)
	}

	// Sort the two lists.
	sort.Ints(pids)
	sort.Ints(tasks)

	// Ensure that two lists are the same.
	if !reflect.DeepEqual(pids, tasks) {
		Fatalf(t, "Lists are not equal:\nxpeted=%s\nfound=%s", pids, tasks)
	}
}

func TestCgroup_TasksFiles(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// A valid cgroup that we will use in the tests.
	_, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)

	// Start a sleep process and put it in the cgroup.
	cmd := exec.Command("/bin/sleep", "60")
	if err := cmd.Start(); err != nil {
		Fatalf(t, "Error starting command: %s", err)
	} else if err := cgroup.AddTask(cmd.Process.Pid); err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	// Get the list of files.
	tasksfiles := cgroup.TasksFiles()

	// ensure that the lenghts are the same.
	if len(tasksfiles) != len(defaultCgroups) {
		Fatalf(t, "tasks files size difference.")
	}

	// See if the command we created is in all the tasks files.
	expected := []byte(fmt.Sprintf("%d\n", cmd.Process.Pid))
	for _, file := range tasksfiles {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		} else if !bytes.Equal(b, expected) {
			Fatalf(t, "Unexpected tasks content: %s", string(b))
		}
	}
}

func TestCgroup_Shutdown(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// ------------------
	// Failure Conditions
	// ------------------

	// Test 1: os.Remove error.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)

		defer func() { osRemove = os.Remove }()
		osRemove = func(name string) error {
			return fmt.Errorf("expected error from os.Remove()")
		}

		cgroup := Cgroup{name: "foo"}
		if err := cgroup.Shutdown(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 2: Can not shutdown with processes in the cgroup.
	func() {
		// A valid cgroup that we will use in the tests.
		_, cgroup := MakeUniqueCgroup(t)
		defer CleanupCgroup(t, cgroup)

		// Start a sleep process and put it in the cgroup.
		cmd := exec.Command("/bin/sleep", "60")
		if err := cmd.Start(); err != nil {
			Fatalf(t, "Error starting command: %s", err)
		} else if err := cgroup.AddTask(cmd.Process.Pid); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		// Now attempt to shutdown.
		if err := cgroup.Shutdown(); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// -------
	// Success
	// -------

	// A valid cgroup that we will use in the tests.
	_, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)

	if err := cgroup.Shutdown(); err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}
}

func TestCgroup_SignalAll(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)
	TestRequiresRoot(t)

	// ------------------
	// Failure Conditions
	// ------------------

	// Test 1: Tasks() failure.
	func() {
		defer func(s string) { cgroupsDir = s }(cgroupsDir)
		cgroupsDir = TempDir(t)

		// Reset the ioutilReadFile function.
		defer func() { ioutilReadFile = ioutil.ReadFile }()
		ioutilReadFile = func(name string) ([]byte, error) {
			return nil, fmt.Errorf("expected error from ioutil.ReadFile()")
		}

		cgroup := Cgroup{name: "foo"}
		if _, err := cgroup.SignalAll(syscall.SIGHUP); err == nil {
			Fatalf(t, "Expected error not returned.")
		}
	}()

	// Test 2: Bad return from syscall.Kill
	func() {
		// A valid cgroup that we will use in the tests.
		_, cgroup := MakeUniqueCgroup(t)
		defer CleanupCgroup(t, cgroup)

		// Start a sleep process and put it in the cgroup.
		cmd := exec.Command("/bin/sleep", "60")
		if err := cmd.Start(); err != nil {
			Fatalf(t, "Error starting command: %s", err)
		} else if err := cgroup.AddTask(cmd.Process.Pid); err != nil {
			Fatalf(t, "Unexpected error: %s", err)
		}

		// Reset the ioutilReadFile function.
		defer func() { syscallKill = syscall.Kill }()
		syscallKill = func(pid int, sig syscall.Signal) error {
			return fmt.Errorf("expected error from ioutil.ReadFile()")
		}

		// Now attempt to signal the process which will fail.
		if _, err := cgroup.SignalAll(syscall.SIGHUP); err == nil {
			Fatalf(t, "Expected error not returned: %s", err)
		}
	}()

	// -------
	// Success
	// -------

	// A valid cgroup that we will use in the tests.
	_, cgroup := MakeUniqueCgroup(t)
	defer CleanupCgroup(t, cgroup)

	// Start a sleep process and put it in the cgroup.
	cmd := exec.Command("/bin/sleep", "60")
	if err := cmd.Start(); err != nil {
		Fatalf(t, "Error starting command: %s", err)
	} else if err := cgroup.AddTask(cmd.Process.Pid); err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	}

	// Attempt to send a -1 signal which is used purely to count
	// the processes.
	if n, err := cgroup.SignalAll(syscall.Signal(-1)); err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	} else if n != 1 {
		Fatalf(t, "Unexpected process count: %d", n)
	}

	// Now attempt to signal using signal 0 which does all the
	// signaling, but the kernel doesn't actually send the signal.
	if n, err := cgroup.SignalAll(syscall.Signal(0)); err != nil {
		Fatalf(t, "Unexpected error: %s", err)
	} else if n != 1 {
		Fatalf(t, "Unexpected process count: %d", n)
	}
}

/*

cgroups.go:func (c *Cgroup) SignalAll(signal syscall.Signal) (int, error) {
setup.go:func CheckCgroups() error {

*/
