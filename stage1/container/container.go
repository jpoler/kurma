// Copyright 2015 Apcera Inc. All rights reserved.

package container

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/apcera/kurma/cgroups"
	kschema "github.com/apcera/kurma/schema"
	"github.com/apcera/kurma/stage2/client"
	"github.com/apcera/logray"
	"github.com/apcera/util/envmap"
	"github.com/appc/spec/schema"

	_ "github.com/apcera/kurma/schema"
)

type ContainerState int

const (
	NEW = ContainerState(iota)
	STARTING
	RUNNING
	STOPPING
	STOPPED
	EXITED
)

// Container represents the operation and management of an individual container
// on the current system.
type Container struct {
	manager *Manager
	log     *logray.Logger

	image            *schema.ImageManifest
	pod              *schema.PodManifest
	initialImageFile io.ReadCloser

	cgroup      *cgroups.Cgroup
	directory   string
	environment *envmap.EnvMap

	shuttingDown bool
	state        ContainerState
	mutex        sync.Mutex
}

// Manifest returns the current pod manifest for the App Container
// Specification.
func (container *Container) Manifest() *schema.PodManifest {
	return container.pod
}

// State returns the current operating state of the container.
func (container *Container) State() ContainerState {
	container.mutex.Lock()
	defer container.mutex.Unlock()
	return container.state
}

// isShuttingDown returns whether the container is currently in the state of
// being shut down. This is an internal flag, separate from the State.
func (container *Container) isShuttingDown() bool {
	container.mutex.Lock()
	defer container.mutex.Unlock()
	return container.shuttingDown
}

// start is an internal function which launches and starts the processes within
// the container.
func (container *Container) start() {
	container.mutex.Lock()
	container.state = STARTING
	container.mutex.Unlock()

	// loop over the container startup functions
	for _, f := range containerStartup {
		if err := f(container); err != nil {
			// FIXME more error handling
			container.log.Errorf("startup error: %v", err)
			return
		}
	}

	container.mutex.Lock()
	container.state = RUNNING
	container.mutex.Unlock()
}

// Stop triggers the shutdown of the Container.
func (container *Container) Stop() error {
	container.mutex.Lock()
	container.shuttingDown = true
	container.state = STOPPING
	container.mutex.Unlock()

	// loop over the container stopping functions
	for _, f := range containerStopping {
		if err := f(container); err != nil {
			// FIXME more error handling
			container.log.Errorf("stopping error: %v", err)
			return err
		}
	}

	container.mutex.Lock()
	container.state = STOPPED
	container.mutex.Unlock()
	return nil
}

// ShortName returns a shortened name that can be used to reference the
// Container. It is made of up of the first 8 digits of the container's UUID.
func (container *Container) ShortName() string {
	if container == nil {
		return ""
	} else if len(container.pod.UUID.String()) >= 8 {
		return container.pod.UUID.String()[0:8]
	}
	return container.pod.UUID.String()
}

func (c *Container) Enter(stream *os.File) error {
	launcher := &client.Launcher{
		Environment: c.environment.Strings(),
		Taskfiles:   c.cgroup.TasksFiles(),
		Stdin:       stream,
		Stdout:      stream,
		Stderr:      stream,
		User:        c.image.App.User,
		Group:       c.image.App.Group,
	}

	// Check for a privileged isolator
	if iso := c.image.App.Isolators.GetByName(kschema.HostPrivlegedName); iso != nil {
		if piso, ok := iso.Value().(*kschema.HostPrivileged); ok {
			if *piso {
				launcher.HostPrivileged = true
			}
		}
	}

	// Get a process from the container and copy its namespaces
	tasks, err := c.cgroup.Tasks()
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		return fmt.Errorf("no processes are running inside the container")
	}
	launcher.SetNS(tasks[0])

	p, err := launcher.Run("/bin/bash")
	if err != nil {
		return err
	}
	_, err = p.Wait()
	return err
}
