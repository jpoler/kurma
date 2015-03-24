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

// ensureContainerPathExists ensures that the specified path within the
// container exists. It will create any missing directories and walk the
// filesystem to ensure any portions that are symlinks are resolved. It returns
// the full host path to the directory.
func (c *Container) ensureContainerPathExists(name string) (string, error) {
	parts := strings.Split(name, string(os.PathSeparator))
	resolvedPath := c.stage3Path()
	containerPath := ""

	for _, p := range parts {
		if p == "" {
			continue
		}
		containerPath = filepath.Join(containerPath, p)

		// resolve this segment
		newResolvedPath, err := c.resolveSymlinkDir(containerPath)
		if err != nil {
			if os.IsNotExist(err) {
				// create it if it doesn't exist
				resolvedPath = filepath.Join(resolvedPath, p)
				if err := os.Mkdir(resolvedPath, os.FileMode(0755)); err != nil {
					return "", err
				}
				continue
			}
			return "", err
		}

		// preserve the resolved path for the next iteration
		resolvedPath = newResolvedPath
	}

	return resolvedPath, nil
}

// Resolves a given directory name relative to the container into a directory
// name relative to the instance manager. This will attempt to follow symlinks
// as best as possible, ensuring that the destination stays inside of the
// container directory.
func (c *Container) resolveSymlinkDir(name string) (string, error) {
	// This is used to compare paths to ensure that they are exactly contained
	// completely within s.RootDirectory()
	root := c.stage3Path()
	checkList := func(fn string) (string, bool) {
		fnPath := filepath.Join(root, fn)
		if len(fnPath) < len(root) {
			return "", false
		} else if fnPath == root {
			return fnPath, true
		} else if strings.HasPrefix(fnPath, root+string(os.PathSeparator)) {
			return fnPath, true
		} else {
			return "", false
		}
	}

	// Loop until we have either walked too far, or we resolve the symlink. This
	// protects us from simple symlink loops.
	checkRecurse := func(name string) (string, error) {
		for depth := 0; depth < 64; depth++ {
			// Get the real path for the file.
			if newName, ok := checkList(name); !ok {
				return "", fmt.Errorf("Name resolved to an unsafe path: %s", name)
			} else if fi, err := os.Lstat(newName); err != nil {
				return "", err
			} else if fi.Mode()&os.ModeSymlink != 0 {
				// If the destination is a symlink then we need to resolve it in order to
				// walk down the chain.
				var err error
				if name, err = os.Readlink(newName); err != nil {
					return "", err
				}
				continue
			} else if !fi.IsDir() {
				return "", fmt.Errorf("Resolved path is not a directory: %s", newName)
			} else {
				return newName, nil
			}
		}
		return "", fmt.Errorf("Symlink depth too excessive.")
	}

	// Loop over the portions of the path to see where they resolve to
	containerPath := ""
	parts := strings.Split(name, string(os.PathSeparator))
	for _, p := range parts {
		if p == "" {
			continue
		}
		containerPath = filepath.Join(containerPath, p)
		newName, allowed := checkList(containerPath)
		if !allowed {
			return "", fmt.Errorf("Name resolved to an unsafe path: %s", name)
		}
		fi, err := os.Lstat(newName)
		if err != nil {
			return "", err
		}

		// if the portion is a symlink, we'll read it and then resolve it out
		if fi.Mode()&os.ModeSymlink != 0 {
			name, err := os.Readlink(newName)
			if err != nil {
				return "", err
			}

			// handle if the path is not absolute, such as "../dir" or just "dir".
			if !filepath.IsAbs(name) {
				name = filepath.Join(filepath.Dir(newName), name)
				name = filepath.Clean(name)
			}
			containerPath = strings.Replace(name, root+string(os.PathSeparator), "", -1)

			// recurse the link to check for additional layers of links
			name, err = checkRecurse(containerPath)
			if err != nil {
				return "", err
			}
			containerPath = strings.Replace(name, root+string(os.PathSeparator), "", -1)
		}
	}

	return filepath.Join(root, containerPath), nil
}
