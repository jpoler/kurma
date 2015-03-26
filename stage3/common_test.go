// Copyright 2013-2015 Apcera Inc. All rights reserved.

// +build linux,cgo

package stage3

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/apcera/continuum/instance_manager/container/cgroups"
	"github.com/apcera/continuum/instance_manager/container/initd/clone"
	"github.com/apcera/util/uuid"

	. "github.com/apcera/util/testtool"
)

// -------
// Helpers
// -------

// Makes a common cgroup and ensures that it gets destroyed on test exit.
func makeCgroup(t *testing.T) *cgroups.Cgroup {
	name := fmt.Sprintf("unittesting-%s", uuid.Variant4().String())
	cgroup, err := cgroups.New(name)
	TestExpectSuccess(t, err)
	AddTestFinalizer(func() {
		TestExpectSuccess(t, cgroup.Destroy())
	})
	return cgroup
}

// Gets information about a specific task from /proc.
func taskInfo(
	t *testing.T, task int,
) (cmdline []string, env []string, ppid int, children []int) {
	// Get the command line of the task.
	cmd, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", task))
	TestExpectSuccess(t, err)
	cmdList := bytes.Split(cmd, []byte{0})
	cmdline = make([]string, len(cmdList)-1)
	for i := range cmdline {
		cmdline[i] = string(cmdList[i])
	}

	// Get the environment of the process.
	envData, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/environ", task))
	TestExpectSuccess(t, err)
	envList := bytes.Split(envData, []byte{0})
	env = make([]string, len(envList)-1)
	for i := range env {
		env[i] = string(envList[i])
	}
	// Sort the environment for consistency.
	sort.Strings(env)

	// Get the pid of the parent process.
	stat, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", task))
	TestExpectSuccess(t, err)
	fields := strings.Split(string(stat), " ")
	if len(fields) < 4 {
		Fatalf(t, "Unknown output in /proc/%d/stat: %d", task, string(stat))
	}
	ppid, err = strconv.Atoi(fields[3])
	TestExpectSuccess(t, err)

	// Now get a list of all of this tasks children.
	cldrn, err := ioutil.ReadFile(
		fmt.Sprintf("/proc/%d/task/%d/children", task, task))
	TestExpectSuccess(t, err)
	childrenStrs := strings.Split(string(cldrn), " ")
	children = make([]int, len(childrenStrs)-1)
	for i := range children {
		children[i], err = strconv.Atoi(childrenStrs[i])
		TestExpectSuccess(t, err)
	}

	// Success
	return
}

func waitTask(t *testing.T, cgroup *cgroups.Cgroup, targetCmd []string, wait time.Duration) (targetPid int, taskPids []int) {
	time.Sleep(50 * time.Millisecond)
	// The sleep above should give initd enough time to fork/exec, but we allow
	// up to wait for the target bin name to appear.
	Timeout(t, wait, 5*time.Millisecond, func() bool {
		running, err := cgroup.Tasks()
		if err != nil {
			return false
		}
		matchingBinaries := 0
		for _, pid := range running {
			cmdLine, _, _, _ := taskInfo(t, pid)
			equal := true
			if len(cmdLine) != len(targetCmd) {
				continue
			}
			for i, arg := range targetCmd {
				if cmdLine[i] != arg {
					equal = false
					break
				}
			}
			if equal {
				matchingBinaries++
				targetPid = pid
				taskPids = running
			}
		}
		return matchingBinaries == 1
	})
	return
}

// Starts an instance of initd, returning the socket file, log file, and the
// pid of the running initd process.
func StartInitd(
	t *testing.T,
) (cgroup *cgroups.Cgroup, socket string, log string, pid int) {
	var err error
	TestRequiresRoot(t)
	dir := TempDir(t)
	socket = path.Join(dir, "socket")
	cmd := clone.Command(os.Args[0])
	cmd.NewIPCNamespace = true
	cmd.NewNetworkNamespace = true
	cmd.NewMountNamespace = true
	cmd.NewPidNamespace = true
	cmd.NewUTSNamespace = true
	cmd.Env = []string{
		"INITD_DEBUG=1",
		fmt.Sprintf("INITD_SOCKET=%s", socket),
	}
	cgroup = makeCgroup(t)
	cmd.TasksFiles = cgroup.TasksFiles()

	// open a file for logging.
	log = path.Join(dir, "log")
	flags := os.O_WRONLY | os.O_SYNC | os.O_CREATE | os.O_APPEND
	mode := os.FileMode(0644)
	logFile, err := os.OpenFile(log, flags, mode)
	TestExpectSuccess(t, err)

	// Setup the various file descriptors.
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin, err = os.Open("/dev/null")
	TestExpectSuccess(t, err)
	err = cmd.Start()
	TestExpectSuccess(t, err)
	pid = cmd.PID
	AddTestFinalizer(func() {
		if !t.Failed() {
			return
		}
		fmt.Println("========== INITD LOG ===========")
		data, _ := ioutil.ReadFile(log)
		fmt.Println(string(data))
	})

	// Ensure that the logs file contains the string showing us that it has
	// started properly.
	Timeout(t, 10*time.Second, time.Second/100, func() bool {
		data, err := ioutil.ReadFile(log)
		TestExpectSuccess(t, err)
		return strings.Contains(string(data), "Starting initd.")
	})

	return
}

// Raw request version of MakeRequest.
func RawRequest(sock, data string, timeout time.Duration) (reply string, err error) {

	// Connect to the socket.
	addr, err := net.ResolveUnixAddr("unix", sock)
	if err != nil {
		return "", err
	}
	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	deadline := time.Now().Add(timeout)
	if err := conn.SetWriteDeadline(deadline); err != nil {
		return "", err
	}

	// Do write inline.
	n, errWrite := conn.Write([]byte(data))
	if errWrite == nil && n != len(data) {
		errWrite = fmt.Errorf("Short write.")
		return "", errWrite
	}

	var errRead error
	wg := sync.WaitGroup{}
	wg.Add(1) // Background Read

	// Read from the client in the background.
	go func() {
		defer wg.Done()
		deadline := time.Now().Add(timeout)
		conn.SetReadDeadline(deadline)
		// Read the response from the client.
		var bytes []byte
		bytes, errRead = ioutil.ReadAll(conn)
		reply = string(bytes)
	}()

	wg.Wait()
	switch {
	case errRead != nil:
		err = errRead
	case !strings.HasPrefix(reply, "REQUEST OK\n"):
		err = fmt.Errorf("Request failed: %s", reply)
	}

	return
}

// Writes a given request to the socket and returns the string written
// back via the request. This returns errors rather than failing the test
// as some tests use this to induce failures.
func MakeRequest(
	sock string, data [][]string, timeout time.Duration,
) (reply string, err error) {
	buffer := bytes.NewBuffer(nil)

	// Write the protocol first.
	buffer.WriteString("1\n")

	// Write the length of the data array to the connection.
	buffer.WriteString(fmt.Sprintf("%d\n", len(data)))
	for _, inner := range data {
		buffer.WriteString(fmt.Sprintf("%d\n", len(inner)))
		for _, str := range inner {
			buffer.WriteString(fmt.Sprintf("%d\n", len(str)))
			buffer.WriteString(str)
		}
	}

	return RawRequest(sock, buffer.String(), timeout)
}

// Runs a slew of bad results against the server in parallel in order
// to ensure that all of them fail as expected.
func BadResultsCheck(t *testing.T, tests [][][]string) {
	// Start the initd process.
	_, socket, _, _ := StartInitd(t)

	// Error reply channel.
	errors := make([]string, 0, len(tests))
	for id, test := range tests {
		reply, err := MakeRequest(socket, test, 10*time.Second)
		if err == nil {
			errors = append(errors, fmt.Sprintf(
				"test %d: Expected error not returned.", id))
		} else if reply != "PROTOCOL ERROR\n" {
			errors = append(errors, fmt.Sprintf(
				"test %d: Reply was not 'PROTOCOL ERROR', was '%s' with err [%v]",
				id, reply, err))
		}
	}

	if len(errors) != 0 {
		Fatalf(t, strings.Join(errors, "\n"))
	}
}
