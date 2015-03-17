// Copyright 2015 Apcera Inc. All rights reserved.

package bootstrap

import (
	"os"
	"syscall"
)

// handleMount takes care of creating the mount path and issuing the mount
// syscall for the mount source, location, and fstype.
func handleMount(source, location, fstype, data string) error {
	if err := os.MkdirAll(location, os.FileMode(0755)); err != nil {
		return err
	}
	return syscall.Mount(source, location, fstype, syscall.MS_MGC_VAL, data)
}
