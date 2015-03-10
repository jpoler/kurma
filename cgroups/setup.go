// Copyright 2013-2014 Apcera Inc. All rights reserved.

// +build linux,cgo

package cgroups

import (
	"fmt"
	"os"
	"path"
)

// #include <sys/mount.h>
import "C"

var cgroupsDir string = "/sys/fs/cgroup"

// This is the list of cgroups that should exist for us to use.
var defaultCgroups []string = []string{
	"cpu",
	"cpuacct",
	"devices",
	"memory",
	"blkio",
}

// Verifies that all of the cgroups directories are actually mounted. If one is
// not mounted then an error will be returned, otherwise nil will be returned.
func CheckCgroups() error {
	// Ensure that each of the defaultCgroups exists in '/sys/fs/cgroup'
	for _, name := range defaultCgroups {
		dir := path.Join(cgroupsDir, name)
		_, err := os.Lstat(dir)
		if err != nil {
			return fmt.Errorf("Missing cgroup mount: %s", dir)
		}
	}

	return nil
}

func CgroupsDirPrefix() string {
	return cgroupsDir
}
