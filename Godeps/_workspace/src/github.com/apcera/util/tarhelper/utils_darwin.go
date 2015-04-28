// Copyright 2013 Apcera Inc. All rights reserved.

// +build darwin

package tarhelper

func makedev(major, minor int64) int {
	return int(major)<<24 | int(minor)
}

func majordev(dev int64) int64 {
	return int64((dev >> 24) & 0xff)
}

func minordev(dev int64) int64 {
	return int64(dev & 0xffffff)
}
