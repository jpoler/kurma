// Copyright 2015 Apcera Inc. All rights reserved.

package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apcera/logray"
	"github.com/apcera/util/proc"
)

var (
	// The setup functions that should be run in order to handle setting up the
	// host system to create and manage containers. These functions focus
	// primarily on runtime actions that must be done each time on boot.
	setupFunctions = []func() error{
		createSystemMounts,
		mountCgroups,
	}

	// The logger is set in Run() so that it inherits any default outputs. If it
	// is set in init(), it won't have them.
	log *logray.Logger
)

const (
	// The default location where cgroups should be mounted. This is a constant
	// because it is referenced in multiple functions.
	cgroupsMount = "/sys/fs/cgroup"
)

// Run handles executing the bootstrap setup. This prepares the current host
// environment to run and manage containers. It will return an error if any part
// of the setup fails.
func Run() error {
	log = logray.New()
	log.Info("Running bootstrap")

	for _, f := range setupFunctions {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

// createSystemMounts configured the default mounts for the host. Since kurma is
// running as PID 1, there is no /etc/fstab, therefore it must mount them
// itself.
func createSystemMounts() error {
	// Default mounts to handle on boot. Note that order matters, they should be
	// alphabetical by mount location. Elements are: mount location, source,
	// fstype.
	systemMounts := [][]string{
		[]string{"/dev", "devtmpfs", "devtmpfs"},
		[]string{"/dev/pts", "none", "devpts"},
		[]string{"/proc", "none", "proc"},
		[]string{"/sys", "none", "sysfs"},

		// put cgroups in a tmpfs so we can create the subdirectories
		[]string{cgroupsMount, "none", "tmpfs"},
	}

	log.Info("Creating system mounts")

	// Check if the /proc/mounts file exists to see if there are mounts that
	// already exist. This is primarily to support testing bootstrapping with
	// kurma launched by kurma (yes, meta)
	var existingMounts map[string]*proc.MountPoint
	if _, err := os.Lstat(proc.MountProcFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to check if %q existed: %v", proc.MountProcFile, err)
	} else if os.IsNotExist(err) {
		// really are freshly booted, /proc isn't mounted, so make this blank
		existingMounts = make(map[string]*proc.MountPoint)
	} else {
		// Get existing mount points.
		existingMounts, err = proc.MountPoints()
		if err != nil {
			return fmt.Errorf("failed to read existing mount points: %v", err)
		}
	}

	for _, mount := range systemMounts {
		location, source, fstype := mount[0], mount[1], mount[2]

		// check if it exists
		if _, exists := existingMounts[location]; exists {
			log.Tracef("- skipping %q, already mounted", location)
			continue
		}

		// perform the mount
		log.Tracef("- mounting %q (type %q) to %q", source, fstype, location)
		if err := handleMount(source, location, fstype, ""); err != nil {
			return fmt.Errorf("failed to mount %q: %v", location, err)
		}
	}
	return nil
}

// mountCgroups handles creating the individual cgroup endpoints that are
// necessary.
func mountCgroups() error {
	// Default cgroups to mount and utilize.
	cgroupTypes := []string{
		"blkio",
		"cpu",
		"cpuacct",
		"devices",
		"memory",
	}

	log.Info("Setting up cgroups")

	for _, cgrouptype := range cgroupTypes {
		location := filepath.Join(cgroupsMount, cgrouptype)
		log.Tracef("- mounting cgroup %q to %q", cgrouptype, location)
		if err := handleMount("none", location, "cgroup", cgrouptype); err != nil {
			return fmt.Errorf("failed to mount cgroup %q: %v", cgrouptype, err)
		}
	}
	return nil
}
