// Copyright 2014 Apcera Inc. All rights reserved.

// +build !windows

package tarhelper

import (
	"os"
	"syscall"
)

func osUmask(mask int) {
	syscall.Umask(mask)
}

func osMknod(name string, mode uint32, dev int) error {
	return syscall.Mknod(name, mode, dev)
}

func osDeviceNumbersForFileInfo(fi os.FileInfo) (int64, int64) {
	if sys, ok := fi.Sys().(*syscall.Stat_t); ok {
		return majordev(int64(sys.Rdev)), minordev(int64(sys.Rdev))
	}
	return 0, 0
}

func uidForFileInfo(fi os.FileInfo) int {
	return int(fi.Sys().(*syscall.Stat_t).Uid)
}

func gidForFileInfo(fi os.FileInfo) int {
	return int(fi.Sys().(*syscall.Stat_t).Gid)
}

func linkCountForFileInfo(fi os.FileInfo) uint {
	return uint(fi.Sys().(*syscall.Stat_t).Nlink)
}

func inodeForFileInfo(fi os.FileInfo) uint64 {
	return fi.Sys().(*syscall.Stat_t).Ino
}
