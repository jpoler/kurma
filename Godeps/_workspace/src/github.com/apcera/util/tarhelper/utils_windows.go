// Copyright 2014 Apcera Inc. All rights reserved.

// +build windows

package tarhelper

import (
	"fmt"
	"os"
)

func makedev(major, minor int64) int {
	panic(fmt.Sprintf("no Windows support for making Unix devices [makedev(%d, %d)]", major, minor))
}

func majordev(dev int64) int64 {
	panic(fmt.Sprintf("no Windows support for making Unix devices [majordev(%d)]", dev))
}

func minordev(dev int64) int64 {
	panic(fmt.Sprintf("no Windows support for making Unix devices [minordev(%d)]", dev))
}

func osUmask(mask int) {
	// noop
}

func osMknod(name string, mode uint32, dev int) error {
	return fmt.Errorf("no Windows support to mknod(%q) (mode %d dev %d)", name, mode, dev)
}

func osDeviceNumbersForFileInfo(_ os.FileInfo) (int64, int64) {
	return 0, 0
}

func uidForFileInfo(_ os.FileInfo) int {
	return 0
}

func gidForFileInfo(_ os.FileInfo) int {
	return 0
}

func linkCountForFileInfo(_ os.FileInfo) uint16 {
	return 1
}

func inodeForFileInfo(_ os.FileInfo) uint64 {
	// if our linkCountForFileInfo() can ever return >1 then we will need to
	// provide real data here
	return 1
}
