// Copyright 2015 Apcera Inc. All rights reserved.
//
// This file is based on code from:
//   https://github.com/rancherio/os
//
// Code is licensed under Apache 2.0.
// Copyright (c) 2014-2015 Rancher Labs, Inc.

package util

/*
#cgo LDFLAGS: -lmount -lblkid
#include<blkid/blkid.h>
#include<libmount/libmount.h>
#include<stdlib.h>
*/
import "C"
import "unsafe"

import (
	"errors"
)

func ResolveDevice(spec string) string {
	cSpec := C.CString(spec)
	defer C.free(unsafe.Pointer(cSpec))
	cString := C.blkid_evaluate_spec(cSpec, nil)
	defer C.free(unsafe.Pointer(cString))
	return C.GoString(cString)
}

func GetFsType(device string) (string, error) {
	var ambi *C.int
	cDevice := C.CString(device)
	defer C.free(unsafe.Pointer(cDevice))
	cString := C.mnt_get_fstype(cDevice, ambi, nil)
	defer C.free(unsafe.Pointer(cString))
	if cString != nil {
		return C.GoString(cString), nil
	}
	return "", errors.New("Error while getting fstype")
}

func intToBool(value C.int) bool {
	if value == 0 {
		return false
	}
	return true
}
