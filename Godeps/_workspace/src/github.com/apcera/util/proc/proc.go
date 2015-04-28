// Copyright 2013 Apcera Inc. All rights reserved.

package proc

import (
	"fmt"
	"strconv"
	"strings"
)

type MountPoint struct {
	Dev     string
	Path    string
	Fstype  string
	Options string
	Dump    int
	Fsck    int
}

// This is the location of the proc mount point. Typically this is only
// modified by unit testing.
var MountProcFile string = "/proc/mounts"

// Reads through /proc/mounts and returns the data associated with the mount
// points as a list of MountPoint structures.
func MountPoints() (map[string]*MountPoint, error) {
	mp := make(map[string]*MountPoint, 0)
	var current *MountPoint
	err := ParseSimpleProcFile(
		MountProcFile,
		nil,
		func(line int, index int, elm string) error {
			switch index {
			case 0:
				current = new(MountPoint)
				current.Dev = elm
			case 1:
				if len(elm) > 0 && elm[0] != '/' {
					return fmt.Errorf(
						"Invalid path on lin %d of file %s: %s",
						line, MountProcFile, elm)
				}
				current.Path = elm
				mp[elm] = current
			case 2:
				current.Fstype = elm
			case 3:
				current.Options = elm
			case 4:
				n, err := strconv.ParseUint(elm, 10, 32)
				if err != nil {
					return fmt.Errorf(
						"Error parsing column %d on line %d of file %s: %s",
						index, line, MountProcFile, elm)
				}
				current.Dump = int(n)
			case 5:
				n, err := strconv.ParseUint(elm, 10, 32)
				if err != nil {
					return fmt.Errorf(
						"Error parsing column %d on line %d of file %s: %s",
						index, line, MountProcFile, elm)
				}
				current.Fsck = int(n)
			default:
				return fmt.Errorf(
					"Too many colums on line %d of file %s",
					line, MountProcFile)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return mp, nil
}

// Stores interface statistics that are gleaned from /proc/dev/net.
type InterfaceStat struct {
	Device       string
	RxBytes      uint64
	RxPackets    uint64
	RxErrors     uint64
	RxDrop       uint64
	RxFifo       uint64
	RxFrame      uint64
	RxCompressed uint64
	RxMulticast  uint64
	TxBytes      uint64
	TxPackets    uint64
	TxErrors     uint64
	TxDrop       uint64
	TxFifo       uint64
	TxFrame      uint64
	TxCompressed uint64
	TxMulticast  uint64
}

// The file that stores network device statistics.
var DeviceStatsFile string = "/proc/det/dev"

// Returns the interface statistics as a map keyed off the interface name.
func InterfaceStats() (map[string]InterfaceStat, error) {
	ret := make(map[string]InterfaceStat, 0)
	var current InterfaceStat
	lastline := -1
	lastindex := -1

	lf := func(index int, line string) error {
		if lastline == index && lastindex == 16 {
			ret[current.Device] = current
		}
		current = InterfaceStat{}
		return nil
	}
	el := func(line int, index int, elm string) (err error) {
		switch index {
		case 0:
			current.Device = strings.Split(elm, ":")[0]
		case 1:
			current.RxBytes, err = strconv.ParseUint(elm, 10, 64)
		case 2:
			current.RxPackets, err = strconv.ParseUint(elm, 10, 64)
		case 3:
			current.RxErrors, err = strconv.ParseUint(elm, 10, 64)
		case 4:
			current.RxDrop, err = strconv.ParseUint(elm, 10, 64)
		case 5:
			current.RxFifo, err = strconv.ParseUint(elm, 10, 64)
		case 6:
			current.RxFrame, err = strconv.ParseUint(elm, 10, 64)
		case 7:
			current.RxCompressed, err = strconv.ParseUint(elm, 10, 64)
		case 8:
			current.RxMulticast, err = strconv.ParseUint(elm, 10, 64)
		case 9:
			current.TxBytes, err = strconv.ParseUint(elm, 10, 64)
		case 10:
			current.TxPackets, err = strconv.ParseUint(elm, 10, 64)
		case 11:
			current.TxErrors, err = strconv.ParseUint(elm, 10, 64)
		case 12:
			current.TxDrop, err = strconv.ParseUint(elm, 10, 64)
		case 13:
			current.TxFifo, err = strconv.ParseUint(elm, 10, 64)
		case 14:
			current.TxFrame, err = strconv.ParseUint(elm, 10, 64)
		case 15:
			current.TxCompressed, err = strconv.ParseUint(elm, 10, 64)
		case 16:
			current.TxMulticast, err = strconv.ParseUint(elm, 10, 64)
		}
		lastline = line
		lastindex = index
		return
	}

	// Now actually attempt to parse the config
	if err := ParseSimpleProcFile(DeviceStatsFile, lf, el); err != nil {
		return nil, err
	}

	return ret, nil
}
