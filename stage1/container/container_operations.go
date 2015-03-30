// Copyright 2015 Apcera Inc. All rights reserved.

package container

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/apcera/kurma/schema"
	"github.com/apcera/kurma/stage3/client"
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
		(*Container).startingNetworking,
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

// startingNetworking handles configuring parts of the networking for the
// container, such as configuring its resolv.conf
func (c *Container) startingNetworking() error {
	c.log.Debug("Configuring network for container")

	if _, err := os.Lstat("/etc/resolv.conf"); err == nil {
		etcPath, err := c.ensureContainerPathExists("etc")
		if err != nil {
			return err
		}
		resolvPath := filepath.Join(etcPath, "resolv.conf")

		if _, err := os.Lstat(resolvPath); err == nil {
			if err := os.RemoveAll(resolvPath); err != nil {
				return err
			}
		}

		hf, err := os.Open("/etc/resolv.conf")
		if err != nil {
			return err
		}
		defer hf.Close()

		cf, err := os.Create(resolvPath)
		if err != nil {
			return err
		}
		defer cf.Close()

		if _, err := io.Copy(cf, hf); err != nil {
			return err
		}
	}

	c.log.Debug("Done configuring networking")
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
	c.environment = appenv

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

	// Initialize the stage2 launcher
	launcher := &client.Launcher{
		SocketPath: c.socketPath(),
		Directory:  c.stage3Path(),
		Chroot:     true,
		Cgroup:     c.cgroup,
		Stdout:     stage2Stdout,
		Stderr:     stage2Stdout,
	}

	// Configure which linux namespaces to create
	nsisolators := false
	if iso := c.image.App.Isolators.GetByName(schema.LinuxNamespacesName); iso != nil {
		if niso, ok := iso.Value().(*schema.LinuxNamespaces); ok {
			launcher.NewIPCNamespace = niso.IPC()
			launcher.NewMountNamespace = niso.Mount()
			launcher.NewNetworkNamespace = niso.Net()
			launcher.NewPIDNamespace = niso.PID()
			launcher.NewUserNamespace = niso.User()
			launcher.NewUTSNamespace = niso.UTS()
			nsisolators = true
		}
	}
	if !nsisolators {
		// set some defaults if no namespace isolator was given
		launcher.NewIPCNamespace = true
		launcher.NewMountNamespace = true
		launcher.NewPIDNamespace = true
		launcher.NewUTSNamespace = true
	}

	// Check for a privileged isolator
	if iso := c.image.App.Isolators.GetByName(schema.HostPrivlegedName); iso != nil {
		if piso, ok := iso.Value().(*schema.HostPrivileged); ok {
			if *piso {
				launcher.HostPrivileged = true
			}
		}
	}

	client, err := launcher.Run()
	if err != nil {
		return err
	}
	c.mutex.Lock()
	c.initdClient = client
	c.mutex.Unlock()

	// iterate the command arguments and fill in any potential environment variable references
	envmap := c.environment.Map()
	envfunc := func(env string) string { return envmap[env] }
	cmdargs := make([]string, len(c.image.App.Exec))
	copy(cmdargs, c.image.App.Exec)
	for i, s := range cmdargs {
		cmdargs[i] = os.Expand(s, envfunc)
	}

	c.log.Tracef("Launching application [%q:%q]: %#v", c.image.App.User, c.image.App.Group, cmdargs)
	c.log.Tracef("Application environment: %#v", c.environment.Strings())
	err = client.Start(
		"app", cmdargs, c.environment.Strings(),
		"/app.stdout", "/app.stderr",
		c.image.App.User, c.image.App.Group,
		time.Second*5)
	if err != nil {
		return err
	}

	// Start a goroutine to handle transitioning to the exited state when all
	// processes die.
	go c.watchContainer()

	c.log.Trace("Done starting stage 2.")
	return nil
}

// watchContainer runs in another goroutine to handle transitioning to the
// exited state if all the processes exit inside the container.
func (c *Container) watchContainer() {
	c.log.Trace("Starting waiting routine.")
	defer c.log.Trace("Stopping waiting routine.")

	// There are two goroutines here. The first is one that runs the Wait()
	// on the initClient. Every time wait finishes it means that a process
	// has exited. When that happens we need to trigger a run of the status
	// query to get the current running status of all the named processes.

	// Spawn a status goroutine to check on the state of the container.
	go c.statusRoutine()

	// Check to make sure we're still running. It might happen where the instance
	// begins tearing down right after the wait goroutine was started. It may be
	// possible for container to hit an error and begin tearind down before the
	// wait goroutine gets scheduled by the runtime. When that happens, just
	// return.
	if c.isShuttingDown() {
		return
	}

	// Get the initdClient and ensure it is still set and not closed.
	initdClient := c.getInitdClient()
	if initdClient == nil || initdClient.Stopped() {
		return
	}

	errors := 0
	for errors < 3 {
		if err := initdClient.Wait(0); err != nil {
			if c.isShuttingDown() {
				// If the container is shutting down then this may not matter
				// so we just bail out.
				return
			}

			c.log.Warnf("Error reading from initd socket: %s", err)
			errors += 1
			continue
		}

		// Spawn a status goroutine to check on the state of the container.
		go c.statusRoutine()
	}

	// Error talking to the initd socket.. This means that we should just fail
	// out completely.
	c.log.Errorf(""+
		"Too many errors trying to talk to the initd socket (%d), "+
		"shutting the container down.", errors)
	c.markExited()
}

// statusRoutine will check the status of the initd process in order to figure
// out if any processes have died.
func (c *Container) statusRoutine() {
	c.log.Trace("Checking the status of running processes.")

	// Check to make sure we're still running. It might happen where the instance
	// begins tearing down right after the status goroutine was started. It may be
	// possible for container to hit an error and begin tearind down before the
	// status goroutine gets scheduled by the runtime. When that happens, just
	// return.
	if c.isShuttingDown() {
		return
	}

	// Get the initdClient
	initdClient := c.getInitdClient()
	if initdClient == nil || initdClient.Stopped() {
		return
	}

	// Query the current status of running processes.
	results, err := initdClient.Status(time.Second)
	if err != nil {
		if c.isShuttingDown() {
			// If the container is shutting down then its not really an error
			// to be unable to call Status().
			return
		}

		// Log so we know what is going on.
		c.log.Errorf("Unable to get process statuses: %v", err)

		// If we're unable to retrieve status, we can't track the state of the
		// processes within the container and should fail the instance.
		c.markExited()
		return
	}

	// We want to count the number of running processes. If there are no running
	// processes, then we'll mark it as exited.
	runningCount := 0
	for _, status := range results {
		if status == "running" {
			runningCount++
		}
	}

	if runningCount == 0 {
		c.log.Debugf("There were no running processes in the container, tearing it down, marking exited.")
		c.markExited()
	}

	c.log.Trace("Done checking process status.")
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
