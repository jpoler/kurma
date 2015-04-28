// Copyright 2015 Apcera Inc. All rights reserved.

package container

import (
	"fmt"
	"io"
	"sync"

	"github.com/apcera/kurma/cgroups"
	kschema "github.com/apcera/kurma/schema"
	"github.com/apcera/logray"
	"github.com/apcera/util/uuid"
	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
)

// Options contains settings that are used by the Container Manager and
// Containers running on the host.
type Options struct {
	ParentCgroupName   string
	ContainerDirectory string
	RequiredNamespaces []string
}

// Manager handles the management of the containers running and available on the
// current host.
type Manager struct {
	Log *logray.Logger

	containers     map[string]*Container
	containersLock sync.RWMutex

	cgroup             *cgroups.Cgroup
	directory          string
	requiredNamespaces []string
}

// NewManager creates a new Manager with the provided options. It will ensure
// the manager is setup and ready to create containers with the provided
// configuration.
func NewManager(opts *Options) (*Manager, error) {
	// validate cgroups is properly setup on the host
	if err := cgroups.CheckCgroups(); err != nil {
		return nil, fmt.Errorf("failed to check cgroups: %v", err)
	}

	// create the parent cgroup for all child containers to be in
	cg, err := cgroups.New(opts.ParentCgroupName)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		Log:                logray.New(),
		containers:         make(map[string]*Container),
		directory:          opts.ContainerDirectory,
		cgroup:             cg,
		requiredNamespaces: opts.RequiredNamespaces,
	}
	return m, nil
}

// Validate will ensure that the image manifest provided is valid to be run on
// the system. It will return nil if it is valid, or will return an error if
// something is invalid.
func (manager *Manager) Validate(imageManifest *schema.ImageManifest) error {
	if imageManifest.App == nil {
		return fmt.Errorf("the manifest must specify an App")
	}
	if len(imageManifest.App.Exec) == 0 {
		return fmt.Errorf("the manifest App.Exec must specify a command to run")
	}

	// If the namespaces isolator is specified, validate a minimum set of namespaces
	if iso := imageManifest.App.Isolators.GetByName(kschema.LinuxNamespacesName); iso != nil {
		if niso, ok := iso.Value().(*kschema.LinuxNamespaces); ok {
			checks := map[string]func() bool{
				"ipc":   niso.IPC,
				"mount": niso.Mount,
				"net":   niso.Net,
				"pid":   niso.PID,
				"user":  niso.User,
				"uts":   niso.UTS,
			}
			for _, ns := range manager.requiredNamespaces {
				f, exists := checks[ns]
				if !exists {
					return fmt.Errorf("Internal server error")
				}
				if !f() {
					return fmt.Errorf("the manifest %s isolator must require the %s namespace",
						kschema.LinuxNamespacesName, ns)
				}
			}
		}
	}

	return nil
}

// Create begins launching a container with the provided image manifest and
// reader as the source of the ACI.
func (manager *Manager) Create(
	name string, imageManifest *schema.ImageManifest, image io.ReadCloser,
) (*Container, error) {
	// revalidate the image
	if err := manager.Validate(imageManifest); err != nil {
		return nil, err
	}

	// handle a blank name
	if name == "" {
		name = imageManifest.Name.String()
	}

	// populate the container
	container := &Container{
		manager:          manager,
		log:              manager.Log.Clone(),
		uuid:             uuid.Variant4().String(),
		waitch:           make(chan bool),
		initialImageFile: image,
		image:            imageManifest,
		pod: &schema.PodManifest{
			ACKind:    schema.PodManifestKind,
			ACVersion: schema.AppContainerVersion,
			Apps: schema.AppList([]schema.RuntimeApp{
				schema.RuntimeApp{
					Name: types.ACName(name),
					App:  imageManifest.App,
					Image: schema.RuntimeImage{
						Name:   &imageManifest.Name,
						Labels: imageManifest.Labels,
					},
				},
			}),
		},
	}
	container.log.SetField("container", container.uuid)
	container.log.Debugf("Launching container %s", container.uuid)

	// add it to the manager's map
	manager.containersLock.Lock()
	manager.containers[container.uuid] = container
	manager.containersLock.Unlock()

	// begin the startup sequence
	container.start()

	return container, nil
}

// removes a child container from the Container Manager.
func (manager *Manager) remove(container *Container) {
	manager.containersLock.Lock()
	container.mutex.Lock()
	delete(manager.containers, container.uuid)
	container.mutex.Unlock()
	manager.containersLock.Unlock()
}

// Containers returns a slice of the current containers on the host.
func (manager *Manager) Containers() []*Container {
	manager.containersLock.RLock()
	defer manager.containersLock.RUnlock()
	containers := make([]*Container, 0, len(manager.containers))
	for _, c := range manager.containers {
		containers = append(containers, c)
	}
	return containers
}

// Container returns a specific container matching the provided UUID, or nil if
// a container with the UUID does not exist.
func (manager *Manager) Container(uuid string) *Container {
	manager.containersLock.RLock()
	defer manager.containersLock.RUnlock()
	return manager.containers[uuid]
}
