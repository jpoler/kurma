// Copyright 2015 Apcera Inc. All rights reserved.

package container

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/apcera/kurma/schema"
	"github.com/apcera/util/envmap"
	"github.com/apcera/util/hashutil"
	"github.com/apcera/util/tarhelper"
)

var (
	// These are the functions that will be called in order to handle container
	// spin up.
	containerStartup = []func(*Container) error{
		(*Container).startingBaseDirectories,
		(*Container).startingFilesystem,
		(*Container).startingEnvironment,
		(*Container).startingCgroups,
		(*Container).launchStage2,
	}

	// These are the functions that will be called in order to handle container
	// teardown.
	containerStopping = []func(*Container) error{
		(*Container).stoppingCgroups,
		(*Container).stoppingDirectories,
		(*Container).stoppingrRemoveFromParent,
	}
)

// startingBaseDirectories handles creating the directory to store the container
// filesystem and tracking files.
func (c *Container) startingBaseDirectories() error {
	c.log.Debug("Setting up directories.")

	// This is the top level directory that we will create for this container.
	c.directory = filepath.Join(c.manager.directory, c.ShortName())

	// Make the directories.
	mode := os.FileMode(0755)
	dirs := []string{c.directory}
	if err := mkdirs(dirs, mode, false); err != nil {
		return err
	}

	// Ensure the directories are owned by the uid/gid that is root inside the
	// container
	// if err := chowns(dirs, c.manager.namespaceUidOffset, c.manager.namespaceGidOffset); err != nil {
	// 	return err
	// }

	c.log.Debug("Done setting up directories.")
	return nil
}

// startingFilesystem extracts the provided ACI file into the container
// filesystem.
func (c *Container) startingFilesystem() error {
	c.log.Debug("Setting up stage2 filesystem")

	if c.initialImageFile == nil {
		c.log.Error("Initial image filesystem is nil")
		return fmt.Errorf("initial image filesystem is nil")
	}

	defer func() {
		c.initialImageFile.Close()
		c.initialImageFile = nil
	}()

	// handle reading the sha
	sr := hashutil.NewSha512(c.initialImageFile)

	// untar the file
	tarfile := tarhelper.NewUntar(sr, filepath.Join(c.directory))
	tarfile.PreserveOwners = true
	tarfile.PreservePermissions = true
	tarfile.Compression = tarhelper.DETECT
	tarfile.AbsoluteRoot = c.directory
	if err := tarfile.Extract(); err != nil {
		return fmt.Errorf("failed to extract stage2 image filesystem: %v", err)
	}

	// put the hash on the pod manifest
	for i, app := range c.pod.Apps {
		if app.Image.Name.Equals(c.image.Name) {
			if err := app.Image.ID.Set(fmt.Sprintf("sha512-%s", sr.Sha512())); err != nil {
				return err
			}
			c.pod.Apps[i] = app
		}
	}

	c.log.Debug("Done up stage2 filesystem")
	return nil
}

// startingEnvironment sets up the environment variables for the container.
func (c *Container) startingEnvironment() error {
	c.environment = envmap.NewEnvMap()
	c.environment.Set("TMPDIR", "/tmp")
	c.environment.Set("HOME", "/")

	// Add the application's environment
	appenv := c.environment.NewChild()
	for _, env := range c.image.App.Environment {
		appenv.Set(env.Name, env.Value)
	}

	return nil
}

// startingCgroups creates the cgroup under which the processes within the
// container will belong to.
func (c *Container) startingCgroups() error {
	c.log.Debug("Setting up the cgroup.")

	// Create the cgroup.
	cgroup, err := c.manager.cgroup.New(c.ShortName())
	if err != nil {
		c.log.Debugf("Error setting up the cgroup: %v", err)
		return err
	} else {
		c.cgroup = cgroup
	}

	// FIXME add OOM notification handler

	c.log.Debug("Done setting up cgroup.")
	return nil
}

// Start the initd. This doesn't actually configure it, just starts it so we
// have a process and namespace to work with in the networking side of the
// world.
func (c *Container) launchStage2() error {
	c.log.Debug("Starting stage 2.")

	// Open a log file that all output from the container will be written to
	var err error
	flags := os.O_WRONLY | os.O_APPEND | os.O_CREATE | os.O_EXCL | os.O_TRUNC
	stage2Stdout, err := os.OpenFile(c.stage2LogPath(), flags, os.FileMode(0666))
	if err != nil {
		return err
	}
	defer stage2Stdout.Close()

	// Open /dev/null which is used for stdin.
	stage2Stdin, err := os.OpenFile("/dev/null", os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer stage2Stdin.Close()

	// Generate the uid and gid maps for user namespaces
	// uidmap := fmt.Sprintf("0 %d %d\n", c.manager.namespaceUidOffset, c.manager.namespaceUidMaximum)
	// gidmap := fmt.Sprintf("0 %d %d\n", c.manager.namespaceGidOffset, c.manager.namespaceGidMaximum)

	// Initialize the options that will be passed to spawn the container.
	args := []string{
		// Default parameters to chroot, and the directory
		"--chroot",
		"--directory", c.stage3Path(),

		// Baseline namespaces that are always used.
		"--new-ipc-namespace",
		"--new-uts-namespace",
		"--new-mount-namespace",
		// "--new-network-namespace",
		"--new-pid-namespace",
	}

	// If user namespaces are to be used, then add the parameter to populate it
	// and the uid and gid maps.
	// if !c.manager.unittestingSkipUserNamespace {
	// 	args = append(args, "--new-user-namespace")
	// }
	// args = append(args, "--uidmap", uidmap)
	// args = append(args, "--gidmap", gidmap)

	// Set the file descriptors it should use for stdin/out/err. Note this uses
	// the ExtraFiles on the os/exec below. The file descriptor numbers start from
	// after stderr (2). They are separate from the fd in this process.
	args = append(args, "--stdinfd", "3")
	args = append(args, "--stdoutfd", "4")
	args = append(args, "--stderrfd", "4")

	// Loop and append all the cgroups taskfiles the container should be in.
	for _, f := range c.cgroup.TasksFiles() {
		args = append(args, "--taskfile", f)
	}

	// Handle any environment variables passed to the app
	for _, env := range c.environment.Strings() {
		args = append(args, "--env", env)
	}

	// Check for a privileged isolator
	for _, iso := range c.image.App.Isolators {
		if iso.Name == schema.PrivlegedName {
			if piso, ok := iso.Value().(*schema.Privileged); ok {
				if *piso {
					args = append(args, "--privileged")
				}
			}
		}
	}

	// Pass the user and group, if they're set
	if c.image.App.User != "" {
		args = append(args, "--user", c.image.App.User)
	}
	if c.image.App.Group != "" {
		args = append(args, "--group", c.image.App.Group)
	}

	// Setup the command line to have it invoke the container's process.
	args = append(args, "--")
	args = append(args, c.image.App.Exec...)

	// Create and initialize the spawnwer.
	cmd := exec.Command(os.Args[0], args...)
	cmd.ExtraFiles = []*os.File{
		stage2Stdin,
		stage2Stdout,
	}
	cmd.Stdin = stage2Stdin
	cmd.Stdout = stage2Stdout
	cmd.Stderr = stage2Stdout

	// The spawner keys off this environment variable to know when it is supposed
	// to run and take over execution.
	cmd.Env = []string{
		"SPAWNER_INTERCEPT=1",
	}

	// Start the container.
	if err := cmd.Start(); err != nil {
		return err
	}
	c.log.Tracef("Spawner PID: %d", cmd.Process.Pid)

	// Start the wait in a goroutine, it will handle reaping the process when it
	// closes.
	go c.watchContainer(cmd)

	c.log.Debug("Done starting stage 2.")
	return nil
}

// watchContainer runs in another goroutine to handle reaping the process when
// the container shuts down, and also to handle transitioning to the exited
// state if the process exits outside of a container shutdown.
func (c *Container) watchContainer(cmd *exec.Cmd) {
	// wait for the process to exit
	cmd.Wait()

	// if it exits, check if the container is shutting down
	if c.isShuttingDown() {
		return
	}

	// if it is still "running", move it to the exited state
	c.mutex.Lock()
	c.state = EXITED
	c.mutex.Unlock()
}

// stoppingCgroups handles terminating all of the processes belonging to the
// current container's cgroup and then deleting the cgroup itself.
func (c *Container) stoppingCgroups() error {
	c.log.Trace("Tearing down cgroups containers.")

	if c.cgroup == nil {
		//  Do nothing, the cgroup was never setup in the first place.
	} else if d, err := c.cgroup.Destroyed(); err != nil {
		return err
	} else if d == false {
		// Now loop through trying to kill all children in the container. This
		// may end up competing with the kernel's zap task. This may take a
		// short period of time so we make sure to induce a very short sleep
		// between iterations.
		for duration := 10 * time.Millisecond; true; duration *= 2 {
			_, err := c.cgroup.SignalAll(syscall.SIGKILL)
			if err != nil {
				return fmt.Errorf("error killing processes: %s", err)
			} else if tasks, _ := c.cgroup.Tasks(); len(tasks) < 2 {
				// No processes killed. The container has no processes
				// running inside of it (including the initd process).
				// It should now be safe to shut it down.
				break
			}

			// Once we send SIGKILL to all processes it will take a small
			// amount of time for parents to be notified of children's
			// death, and for all the various resource cleanup to happen.
			// Since we don't have a callback for when that is complete we
			// sleep here a very small amount of time before we try again.
			// Each iteration we increase the sleep so that we don't almost
			// busy loop the host OS.
			time.Sleep(duration)
		}

		// So the cgroup should no longer have members. Because of this we can
		// Destroy it safely.
		if err := c.cgroup.Destroy(); err != nil {
			return err
		}
	}

	// Make sure future calls don't attempt destruction.
	c.mutex.Lock()
	c.cgroup = nil
	c.mutex.Unlock()

	c.log.Trace("Done tearing down cgroups containers.")
	return nil
}

// stoppingDirectories removes the directories associated with this Container.
func (c *Container) stoppingDirectories() error {
	c.log.Trace("Removing container directories.")

	// If a directory has not been assigned then bail out
	// early.
	if c.directory == "" {
		return nil
	}

	if err := unmountDirectories(c.directory); err != nil {
		c.log.Warnf("failed to unmount container directories: %s", err)
		return err
	}

	// Remove the directory that was created for this container, unless it is
	// specified to keep it.
	if err := os.RemoveAll(c.directory); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	c.log.Trace("Done tearing down container directories.")
	return nil
}

// stoppingrRemoveFromParent removes the container object itself from the
// Container Manager.
func (c *Container) stoppingrRemoveFromParent() error {
	c.log.Trace("Removing from the Container Manager.")
	c.manager.remove(c)
	return nil
}
