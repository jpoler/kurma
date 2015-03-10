// Copyright 2015 Apcera Inc. All rights reserved.

package container

import (
	"io"
	"sync"

	"github.com/apcera/kurma/cgroups"
	"github.com/apcera/logray"
	"github.com/apcera/util/envmap"
	"github.com/appc/spec/schema"
)

type ContainerState int

const (
	NEW = ContainerState(iota)
	STARTING
	RUNNING
	STOPPING
	STOPPED
	FINISHED
)

// Container represents the operation and management of an individual container
// on the current system.
type Container struct {
	manager *Manager
	log     *logray.Logger

	image            *schema.ImageManifest
	manifest         *schema.ContainerRuntimeManifest
	initialImageFile io.ReadCloser

	cgroup      *cgroups.Cgroup
	directory   string
	environment *envmap.EnvMap

	shuttingDown bool
	state        ContainerState
	mutex        sync.Mutex
}

// Manifest returns the current container manifest for the App Container
// Specification.
func (container *Container) Manifest() *schema.ContainerRuntimeManifest {
	return container.manifest
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
	} else if len(container.manifest.UUID.String()) >= 8 {
		return container.manifest.UUID.String()[0:8]
	}
	return container.manifest.UUID.String()
}
