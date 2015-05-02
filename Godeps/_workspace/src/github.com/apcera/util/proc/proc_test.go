// Copyright 2013-2015 Apcera Inc. All rights reserved.

package proc

import (
	"strings"
	"testing"

	. "github.com/apcera/util/testtool"
)

func TestMountPoints(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)

	// Test 1: Very basic /proc/mounts file. Ensure that each
	//         field is properly parsed, the order is correct, etc.
	MountProcFile = WriteTempFile(t, strings.Join([]string{
		"rootfs1 / rootfs2 rw 0 0",
	}, "\n"))
	if mp, err := MountPoints(); err != nil {
		Fatalf(t, "Error from MountPoints: %s", err)
	} else if len(mp) != 1 {
		Fatalf(t, "Bad return value: %#v", mp)
	} else if mp["/"].Dev != "rootfs1" {
		Fatalf(t, "invalid device: %s", mp["/"].Dev)
	} else if mp["/"].Path != "/" {
		Fatalf(t, "invalid path: %s", mp["/"].Path)
	} else if mp["/"].Fstype != "rootfs2" {
		Fatalf(t, "invalid file system type: %s", mp["/"].Fstype)
	} else if mp["/"].Options != "rw" {
		Fatalf(t, "invalid options: %s", mp["/"].Options)
	} else if mp["/"].Dump != 0 {
		Fatalf(t, "invalid dump value: %d", mp["/"].Dump)
	} else if mp["/"].Fsck != 0 {
		Fatalf(t, "invalid fsck value: %d", mp["/"].Fsck)
	}

	// Test 2: Priority, two mounts in the same path. Ensure that
	//         the last listed always wins.
	MountProcFile = WriteTempFile(t, strings.Join([]string{
		"bad / bad bad 1 1",
		"rootfs1 / rootfs2 rw 0 0",
	}, "\n"))
	if mp, err := MountPoints(); err != nil {
		Fatalf(t, "Error from MountPoints: %s", err)
	} else if len(mp) != 1 {
		Fatalf(t, "Bad return value: %#v", mp)
	} else if mp["/"].Dev != "rootfs1" {
		Fatalf(t, "invalid device: %s", mp["/"].Dev)
	} else if mp["/"].Path != "/" {
		Fatalf(t, "invalid path: %s", mp["/"].Path)
	} else if mp["/"].Fstype != "rootfs2" {
		Fatalf(t, "invalid file system type: %s", mp["/"].Fstype)
	} else if mp["/"].Options != "rw" {
		Fatalf(t, "invalid options: %s", mp["/"].Options)
	} else if mp["/"].Dump != 0 {
		Fatalf(t, "invalid dump value: %d", mp["/"].Dump)
	} else if mp["/"].Fsck != 0 {
		Fatalf(t, "invalid fsck value: %d", mp["/"].Fsck)
	}

	// Test 3: Bad path value (relative or otherwise invalid.)
	MountProcFile = WriteTempFile(t, strings.Join([]string{
		"dev badpath fstype options 0 0",
	}, "\n"))
	if _, err := MountPoints(); err == nil {
		Fatalf(t, "Expected an error from MountPoints()")
	}

	// Test 4: Bad dump value (not an int)
	MountProcFile = WriteTempFile(t, strings.Join([]string{
		"dev / fstype options bad 0",
	}, "\n"))
	if _, err := MountPoints(); err == nil {
		Fatalf(t, "Expected an error from MountPoints()")
	}

	// Test 5: Bad dump value (negative)
	MountProcFile = WriteTempFile(t, strings.Join([]string{
		"dev / fstype options -1 0",
	}, "\n"))
	if _, err := MountPoints(); err == nil {
		Fatalf(t, "Expected an error from MountPoints()")
	}

	// Test 6: Bad dump value (not an int)
	MountProcFile = WriteTempFile(t, strings.Join([]string{
		"dev / fstype options 0 bad",
	}, "\n"))
	if _, err := MountPoints(); err == nil {
		Fatalf(t, "Expected an error from MountPoints()")
	}

	// Test 7: Bad dump value (negative)
	MountProcFile = WriteTempFile(t, strings.Join([]string{
		"dev / fstype options 0 -1",
	}, "\n"))
	if _, err := MountPoints(); err == nil {
		Fatalf(t, "Expected an error from MountPoints()")
	}

	// Test 8: Too many columns.
	MountProcFile = WriteTempFile(t, strings.Join([]string{
		"dev / fstype options 0 0 extra",
	}, "\n"))
	if _, err := MountPoints(); err == nil {
		Fatalf(t, "Expected an error from MountPoints()")
	}
}

func TestInterfaceStats(t *testing.T) {
	StartTest(t)
	defer FinishTest(t)

	expect := func(expected InterfaceStat, returned InterfaceStat) {
		if expected.Device != returned.Device {
			Fatalf(t, "Expected Device=%d, got %d",
				expected.Device, returned.Device)
		} else if expected.RxBytes != returned.RxBytes {
			Fatalf(t, "Expected RxBytes=%d, got %d",
				expected.RxBytes, returned.RxBytes)
		} else if expected.RxPackets != returned.RxPackets {
			Fatalf(t, "Expected RxPackets=%d, got %d",
				expected.RxPackets, returned.RxPackets)
		} else if expected.RxErrors != returned.RxErrors {
			Fatalf(t, "Expected RxErrors=%d, got %d",
				expected.RxErrors, returned.RxErrors)
		} else if expected.RxDrop != returned.RxDrop {
			Fatalf(t, "Expected RxDrop=%d, got %d",
				expected.RxDrop, returned.RxDrop)
		} else if expected.RxFifo != returned.RxFifo {
			Fatalf(t, "Expected RxFifo=%d, got %d",
				expected.RxFifo, returned.RxFifo)
		} else if expected.RxFrame != returned.RxFrame {
			Fatalf(t, "Expected RxFrame=%d, got %d",
				expected.RxFrame, returned.RxFrame)
		} else if expected.RxCompressed != returned.RxCompressed {
			Fatalf(t, "Expected RxCompressed=%d, got %d",
				expected.RxCompressed, returned.RxCompressed)
		} else if expected.RxMulticast != returned.RxMulticast {
			Fatalf(t, "Expected RxMulticast=%d, got %d",
				expected.RxMulticast, returned.RxMulticast)
		} else if expected.TxBytes != returned.TxBytes {
			Fatalf(t, "Expected TxBytes=%d, got %d",
				expected.TxBytes, returned.TxBytes)
		} else if expected.TxPackets != returned.TxPackets {
			Fatalf(t, "Expected TxPackets=%d, got %d",
				expected.TxPackets, returned.TxPackets)
		} else if expected.TxErrors != returned.TxErrors {
			Fatalf(t, "Expected TxErrors=%d, got %d",
				expected.TxErrors, returned.TxErrors)
		} else if expected.TxDrop != returned.TxDrop {
			Fatalf(t, "Expected TxDrop=%d, got %d",
				expected.TxDrop, returned.TxDrop)
		} else if expected.TxFifo != returned.TxFifo {
			Fatalf(t, "Expected TxFifo=%d, got %d",
				expected.TxFifo, returned.TxFifo)
		} else if expected.TxFrame != returned.TxFrame {
			Fatalf(t, "Expected TxFrame=%d, got %d",
				expected.TxFrame, returned.TxFrame)
		} else if expected.TxCompressed != returned.TxCompressed {
			Fatalf(t, "Expected TxCompressed=%d, got %d",
				expected.TxCompressed, returned.TxCompressed)
		} else if expected.TxMulticast != returned.TxMulticast {
			Fatalf(t, "Expected TxMulticast=%d, got %d",
				expected.TxMulticast, returned.TxMulticast)
		}
	}

	// -----------------------------
	// Test 1: Real simple use case.
	// -----------------------------

	DeviceStatsFile = WriteTempFile(t, strings.Join([]string{
		"header 1",
		"header 2",
		"dev0: 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16",
	}, "\n"))
	expected := InterfaceStat{
		Device:       "dev0",
		RxBytes:      1,
		RxPackets:    2,
		RxErrors:     3,
		RxDrop:       4,
		RxFifo:       5,
		RxFrame:      6,
		RxCompressed: 7,
		RxMulticast:  8,
		TxBytes:      9,
		TxPackets:    10,
		TxErrors:     11,
		TxDrop:       12,
		TxFifo:       13,
		TxFrame:      14,
		TxCompressed: 15,
		TxMulticast:  16,
	}
	if stats, err := InterfaceStats(); err != nil {
		Fatalf(t, "Error from TestInterfaceStats: %s", err)
	} else if len(stats) != 1 {
		Fatalf(t, "Bad return value: %#v", stats)
	} else {
		expect(expected, stats["dev0"])
	}

	// -----------------------------
	// Test 2: Invalid format
	// -----------------------------

	DeviceStatsFile = WriteTempFile(t, strings.Join([]string{
		"header 1",
		"header 2",
		"dev0: NaN NaN NaN NaN NaN NaN NaN NaN NaN NaN NaN NaN NaN NaN NaN NaN",
	}, "\n"))
	if _, err := InterfaceStats(); err == nil {
		Fatalf(t, "Expected error not returned.")
	}
}
