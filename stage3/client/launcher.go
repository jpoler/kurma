// Copyright 2015 Apcera Inc. All rights reserved.

package client

import (
	"fmt"
	"os"
	"time"

	kclient "github.com/apcera/kurma/stage2/client"
	"github.com/apcera/kurma/util/cgroups"

	_ "github.com/apcera/kurma/stage3"
)

// DefaultChrootPath defines the default location where the Chroot call should
// be made to fully isolate the container filesystem.
const DefaultChrootPath = "/tmp/container"

// DefaultTimeout is the amount of time that is allowed for the init process to
// create its socket after launching.
var DefaultTimeout = time.Second * 10

// Launcher handles wrapping stage2 to execute the stage3 init process. It has
// many of the same settings which gets propagated down to stage2.
type Launcher struct {
	SocketPath string
	Directory  string
	Uidmap     string
	Gidmap     string

	NewIPCNamespace     bool
	NewMountNamespace   bool
	NewNetworkNamespace bool
	NewPIDNamespace     bool
	NewUTSNamespace     bool
	NewUserNamespace    bool

	HostPrivileged bool
	Chroot         bool

	Cgroup *cgroups.Cgroup

	Stdin  *os.File
	Stdout *os.File
	Stderr *os.File
}

// Run will instantiate the container and execute the init process within it. It
// will return the client to talk to the init, or an error in the case of a
// failure.
func (l *Launcher) Run() (Client, error) {
	// Initialize the stage2 launcher
	launcher := &kclient.Launcher{
		Directory:     l.Directory,
		BindDirectory: DefaultChrootPath,
		// Our Chroot flag is used against the Client.Chroot() call.
		Chroot:              false,
		Detach:              true,
		Stdin:               l.Stdin,
		Stdout:              l.Stdout,
		Stderr:              l.Stderr,
		Uidmap:              l.Uidmap,
		Gidmap:              l.Gidmap,
		HostPrivileged:      l.HostPrivileged,
		NewIPCNamespace:     l.NewIPCNamespace,
		NewMountNamespace:   l.NewMountNamespace,
		NewNetworkNamespace: l.NewNetworkNamespace,
		NewPIDNamespace:     l.NewPIDNamespace,
		NewUTSNamespace:     l.NewUTSNamespace,
		NewUserNamespace:    l.NewUserNamespace,
		Taskfiles:           l.Cgroup.TasksFiles(),
		Environment: []string{
			"INITD_INTERCEPT=1",
			fmt.Sprintf("INITD_SOCKET=%s", l.SocketPath),
		},
	}

	// get the executable path to ourself
	self, err := os.Readlink("/proc/self/exe")
	if err != nil {
		return nil, err
	}

	if _, err := launcher.Run(self); err != nil {
		return nil, err
	}

	// create the socket and wait for it
	c := New(l.SocketPath)
	if err := c.WaitForSocket(DefaultTimeout); err != nil {
		return nil, err
	}

	// if the Chroot flag was set, then have the init process chroot itself.
	if l.Chroot {
		if err := c.Chroot(DefaultChrootPath, l.HostPrivileged, DefaultTimeout); err != nil {
			return nil, fmt.Errorf("failed to chroot the container: %v", err)
		}
	}

	return c, nil
}
