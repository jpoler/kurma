// Copyright 2013 Apcera Inc. All rights reserved.

// +build linux

package tarhelper

func makedev(major, minor int64) int {
	return int(major)<<8 | int(minor)
}

func majordev(dev int64) int64 {
	return int64((dev >> 8) & 0xff)
}

func minordev(dev int64) int64 {
	return int64(dev & 0xff)
}
