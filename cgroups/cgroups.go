// Copyright 2013 Apcera Inc. All rights reserved.

// +build linux,cgo

package cgroups

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/apcera/util/proc"
)

const (
	cpuPeriod = "cpu.cfs_period_us"
	cpuQuota  = "cpu.cfs_quota_us"
	memLimit  = "memory.limit_in_bytes"
	memUsage  = "memory.usage_in_bytes"
)

// ------------------------
// Helpers for Unit Testing
// ------------------------

var ioutilReadFile func(string) ([]byte, error) = ioutil.ReadFile
var osLstat func(string) (os.FileInfo, error) = os.Lstat
var osMkdir func(string, os.FileMode) error = os.Mkdir
var osRemove func(string) error = os.Remove
var syscallKill func(int, syscall.Signal) error = syscall.Kill

// ----------------
// Cgroup structure
// ----------------

type Cgroup struct {
	// Name is the name of this cgroup, which is actually a path under the cgroups
	// mount point.
	name string
}

// Creates a new Cgroup on the system. this will make a directory in each of the
// defaultCgroups subdirectories named for the given name.
func New(name string) (*Cgroup, error) {
	c := new(Cgroup)
	c.name = name

	makeChildDirectory := func(dir string) error {
		// Attempt to make the directory.
		if stat, err := osLstat(dir); err == nil {
			if !stat.IsDir() {
				return fmt.Errorf("Cgroup directory %s is not a directory.", dir)
			}
		} else if os.IsNotExist(err) {
			// Make the directory which should in turn make all the various files in
			// the directory that we will use to control the cgroup.
			if err := osMkdir(dir, 0755); err != nil {
				return fmt.Errorf("Error making cgroup directory %s: %s", dir, err)
			}
		} else {
			return err
		}

		// Ensure that the mount point was created properly by checking for the
		// 'tasks' file. If it doesn't exist, or has anything in it then this
		// creation failed.
		tasksFile := path.Join(dir, "tasks")
		if b, err := ioutilReadFile(tasksFile); err != nil {
			return fmt.Errorf("Error reading cgroup 'tasks' file %s: %s", tasksFile, err)
		} else if len(b) != 0 {
			return fmt.Errorf("New cgroup already has tasks: %s", strings.Replace(string(b), "\n", ",", -1))
		}

		return nil
	}

	// Loop through each of the default cgroup types attempting to make the child
	// directory.
	for _, cgroup := range defaultCgroups {
		dir := path.Join(cgroupsDir, cgroup, name)
		if err := makeChildDirectory(dir); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Simple function to recover a cgroup from a stored state.
func (c *Cgroup) Recover(name string) *Cgroup {
	return &Cgroup{name: path.Join(c.name, name)}
}

// Loop through all the groups in defaultCgroups and call the given function on
// the directory.
func (c *Cgroup) forEach(f func(string) error) error {
	for _, ctype := range defaultCgroups {
		dir := path.Join(cgroupsDir, ctype, c.name)
		if err := f(dir); err != nil {
			return err
		}
	}
	return nil
}

// Add the given process (pid) to the container. If this returns an error then
// the pid may be in some, but not all containers.
func (c *Cgroup) AddTask(pid int) error {
	addTask := func(dir string) error {
		flags := os.O_WRONLY | os.O_APPEND
		tasksfile := path.Join(dir, "tasks")
		f, err := os.OpenFile(tasksfile, flags, os.FileMode(0644))
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString(fmt.Sprintf("%d\n", pid))
		if err != nil {
			return err
		}
		return nil
	}

	// Walk through each cgroup type in defaultCgroups
	if err := c.forEach(addTask); err != nil {
		return err
	}

	return nil
}

// CPUUsed returns the total amount of cpu used by processes in the cgroup.
func (c *Cgroup) CPUUsed() (i int64, err error) {
	return proc.ReadInt64(filepath.Join(cgroupsDir, "cpuacct", c.name, "cpuacct.usage"))
}

// Returns cgroup objects for each of the children in this cgroup. Note that
// this will return a child object if the cgroup only exists in one of the
// default cgroup types.
func (c *Cgroup) Children() ([]*Cgroup, error) {
	children := make(map[string]*Cgroup)

	getChildren := func(dir string) error {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		// Make an array for all the children.
		for i := range files {
			if files[i].IsDir() {
				cgroup := new(Cgroup)
				cgroup.name = path.Join(c.name, files[i].Name())
				children[cgroup.name] = cgroup
			}
		}

		// Success.
		return nil
	}

	// Walk each of our cgroups types and add the children into the map.
	if err := c.forEach(getChildren); err != nil {
		return nil, err
	}

	// Convert the map into an array.
	childrenList := make([]*Cgroup, 0, len(children))
	for _, v := range children {
		childrenList = append(childrenList, v)
	}

	return childrenList, nil
}

// This call will destroy the cgroup by force. Removing all associated
// directories and ensuring that no cgroup by that name exists anymore.  This
// will kill all processes within the cgroup as well as remove its memory and
// CPU allocations. Note that on error the cgroups setup will remain in an
// inconsistent state. This call is idempotent though so it is safe to call
// again.
func (c *Cgroup) Destroy() error {
	// Destroy all children as well
	children, err := c.Children()
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := child.Destroy(); err != nil {
			return err
		}
	}

	// Loop while there are still processes in the container killing them.
	for {
		n, err := c.SignalAll(syscall.SIGKILL)
		if err != nil {
			return err
		} else if n == 0 {
			break
		}
		// Sleep just a short bit to ensure that we don't block the golang
		// scheduler.
		time.Sleep(time.Millisecond)
	}

	// Attempt to shut it down.
	if err := c.Shutdown(); err != nil {
		return err
	}

	return nil
}

// Checks to see if this cgroup has already been destroyed. If this is true then
// its directory is removed and it is completely shut down.
func (c *Cgroup) Destroyed() (bool, error) {
	for _, cgroup := range defaultCgroups {
		dir := path.Join(cgroupsDir, cgroup, c.name)
		if _, err := osLstat(dir); err == nil {
			return false, nil
		} else if !os.IsNotExist(err) {
			return false, err
		}
	}

	// None of the directories exist, return success.
	return true, nil
}

// LimitCPU sets the CPU utilization allowance for this container in ms/sec. A
// value over 1000 grants access to more than one CPU.
func (c *Cgroup) LimitCPU(limit int64) error {
	// Period of child containers should match the period of top-level cgroup or
	// jobs won't start.
	defaultPeriod, err := ioutil.ReadFile(filepath.Join(cgroupsDir, "cpu", cpuPeriod))
	if err != nil {
		return err
	}
	defaultPeriod = bytes.Trim(defaultPeriod, "\n")

	period, err := strconv.Atoi(string(defaultPeriod))
	if err != nil {
		return err
	}

	fn := filepath.Join(cgroupsDir, "cpu", c.name, cpuPeriod)
	if err := ioutil.WriteFile(fn, defaultPeriod, 0644); err != nil {
		return err
	}

	fn = filepath.Join(cgroupsDir, "cpu", c.name, cpuQuota)

	// Limit comes in as ms/s, so we need to scale with respect to period.
	// (e.g. 100 ms/s with a period of 100000 us -> quota of 10000 us)
	quota := limit * int64(period/1000)
	quotaString := strconv.FormatInt(quota, 10)

	if err := ioutil.WriteFile(fn, []byte(quotaString), 0644); err != nil {
		return err
	}

	return nil
}

// Functionally limits memory use within this container. This will impact all
// processes in this container (if there are any) and may fail if a process is
// already using more memory than this limit. Its highly advisable that this
// limit be set before any processes are created and then only ever grown.
func (c *Cgroup) LimitMemory(limit int64) error {
	limitBytes := []byte(fmt.Sprintf("%d", limit))

	fn := filepath.Join(cgroupsDir, "memory", c.name, memLimit)
	if err := ioutil.WriteFile(fn, limitBytes, 0644); err != nil {
		return err
	}

	return nil
}

// MemoryUsed returns the total number of bytes used by processes in the cgroup.
func (c *Cgroup) MemoryUsed() (int64, error) {
	return proc.ReadInt64(filepath.Join(cgroupsDir, "memory", c.name, "memory.usage_in_bytes"))
}

// Returns the total number of bytes used in the container for disk.  Keys off
// of the directory path to the container's root directory.  This runs as IM
// (root) from outside the container itself.  Since the container root is an LVM
// volume we can stat the volume to calculate the usage Returns -1 for the byte
// count if returning an error
func (c *Cgroup) DiskUsed(rootDir string) (int64, error) {
	syscallRes := syscall.Statfs_t{}
	err := syscall.Statfs(rootDir, &syscallRes)

	if err != nil {
		return -1, err
	}

	usedBlocks := syscallRes.Blocks - syscallRes.Bfree

	// verify we're not overflowing an int64 before we convert
	uint64UsedBytes := usedBlocks * uint64(syscallRes.Bsize)
	if uint64UsedBytes > uint64(math.MaxInt64) {
		return -1, fmt.Errorf("Error calculating container size for %q, larger than MaxInt64", rootDir)
	}

	return int64(uint64UsedBytes), nil

}

// Creates a new Cgroup that will be a child of the given cgroup.
func (c *Cgroup) New(name string) (*Cgroup, error) {
	return New(path.Join(c.name, name))
}

// Returns the name given to this context.
func (c *Cgroup) Name() string {
	return c.name
}

// Returns a list of all the tasks in the cgroup. Note that this does not
// include any child cgroups. This also includes tasks, which are thread pids as
// well as process pids.
func (c *Cgroup) Tasks() ([]int, error) {
	m := make(map[uint64]bool)
	r := make([]int, 0)

	// Gathers all the tasks in the cgroup, checks to see if we have already added
	// them and if not then add them to r (the list we will return)
	tasks := func(dir string) error {
		if b, err := ioutilReadFile(path.Join(dir, "tasks")); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		} else {
			barr := bytes.Split(b, []byte{'\n'})
			for _, line := range barr {
				pid64, err := strconv.ParseUint(string(line), 10, 31)
				if err != nil {
					// Empty lines are not errors, but lines with content that can not be
					// parsed as a uint are errors in this file.
					if len(line) != 0 {
						return err
					}
				} else if _, ok := m[pid64]; !ok {
					r = append(r, int(pid64))
					m[pid64] = true
				}
			}
		}

		return nil
	}

	// Loop through each cgroup type and gather the tasks.
	if err := c.forEach(tasks); err != nil {
		return nil, err
	}

	removeDuplicates := func(tasks []int) []int {
		found := make(map[int]bool)
		newTasks := []int{}
		for _, task := range tasks {
			if !found[task] {
				newTasks = append(newTasks, task)
				found[task] = true
			}
		}

		return newTasks
	}

	return removeDuplicates(r), nil
}

// Returns the list of tasks files that need to be modified in order to be an
// active member of all the various containers.
func (c *Cgroup) TasksFiles() []string {
	out := make([]string, len(defaultCgroups))
	for i, cgroup := range defaultCgroups {
		out[i] = path.Join(cgroupsDir, cgroup, c.name, "tasks")
	}
	return out
}

// This will attempt to destroy the cgroup. If there are currently processes
// included in this group then this call will fail. It is the callers
// responsibility to ensure that all processes have been killed and waited on
// before calling this.
func (c *Cgroup) Shutdown() error {
	shutdown := func(dir string) error {
		if err := osRemove(dir); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		return nil
	}

	// Walk through each container type shutting it down.
	if err := c.forEach(shutdown); err != nil {
		return err
	}

	// Success.
	return nil
}

// Kills all processes in this container using the given signal. This will walk
// through all the tasks in this group sending them the given signal.  This will
// return the number of tasks signaled. In the event of an error the number of
// tasks signaled will still be returned however it might not match the number
// of tasks in the cgroup.
func (c *Cgroup) SignalAll(signal syscall.Signal) (int, error) {
	tasks, err := c.Tasks()
	if err != nil {
		return -1, err
	}

	// Sort tasks in reverse PID order so that the initd process is normally the
	// last process killed. This may help alleviate issues we have seen with
	// processes holding on to files inside the container during cleanup.
	sort.Sort(sort.Reverse(sort.IntSlice(tasks)))

	// Walk through and signal each task.
	signaled := 0
	for _, task := range tasks {
		if signal == -1 {
			signaled += 1
			// Do nothing.
		} else if err := syscallKill(task, signal); err != nil {
			// In the case that this is a "process not found" error we actually just
			// ignore the error since this can happen as processes are
			// starting/stopping in the container.
			if err != syscall.ESRCH {
				return signaled, fmt.Errorf("Can not kill process %d: %s", task, err)
			}
		} else {
			signaled += 1
		}
		time.Sleep(time.Millisecond * 50)
	}

	return signaled, err
}
