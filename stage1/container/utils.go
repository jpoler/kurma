// Copyright 2015 Apcera Inc. All rights reserved.

package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/apcera/util/proc"
)

func (c *Container) imageManifestPath() string {
	return filepath.Join(c.directory, "manifest")
}

func (c *Container) containerManifestPath() string {
	return filepath.Join(c.directory, "container")
}

func (c *Container) stage2LogPath() string {
	return filepath.Join(c.directory, "stage2.log")
}

func (c *Container) stage3Path() string {
	return filepath.Join(c.directory, "rootfs")
}

func mkdirs(dirs []string, mode os.FileMode, existOk bool) error {
	for i := range dirs {
		// Make sure that this directory doesn't currently exist if existOk
		// is false.
		if stat, err := os.Lstat(dirs[i]); err == nil {
			if !existOk {
				return fmt.Errorf("lstat: path already exists: %s", dirs[i])
			} else if !stat.IsDir() {
				return fmt.Errorf("lstat: %s is not a directory.", dirs[i])
			}
		} else if !os.IsNotExist(err) {
			return err
		} else if err := os.Mkdir(dirs[i], mode); err != nil {
			return fmt.Errorf("mkdir: %s", err)
		}

		// Ensure that the mode is applied by running chmod against it. We
		// need to do this because Mkdir will apply umask which might screw
		// with the permissions.
		if err := os.Chmod(dirs[i], mode); err != nil {
			return fmt.Errorf("chmod: %s", err)
		}
	}
	return nil
}

func chowns(paths []string, uid, gid int) error {
	for _, p := range paths {
		if err := os.Chown(p, uid, gid); err != nil {
			return fmt.Errorf("chown: %q - %v", p, err)
		}
	}
	return nil
}

func unmountDirectories(path string) error {
	// Get the list of mount points that are under this container's directory
	// and then attempt to unmount them in reverse order. This is required
	// so that all mounts are unmounted before a parent is unmounted.
	mountPoints := make([]string, 0, 100)
	root := path + string(os.PathSeparator)
	err := proc.ParseSimpleProcFile(
		proc.MountProcFile,
		nil,
		func(line int, index int, elem string) error {
			switch {
			case index != 1:
			case elem == path:
				mountPoints = append(mountPoints, elem)
			case strings.HasPrefix(elem, root):
				mountPoints = append(mountPoints, elem)
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Now walk the list in reverse order unmounting each point one at a time.
	for i := len(mountPoints) - 1; i >= 0; i-- {
		if err := syscall.Unmount(mountPoints[i], syscall.MNT_FORCE); err != nil {
			return fmt.Errorf("failed to unmount %q: %v", mountPoints[i], err)
		}
	}

	return nil
}
