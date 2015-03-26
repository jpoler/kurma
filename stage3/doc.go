// Copyright 2013-2015 Apcera Inc. All rights reserved.

// +build linux,cgo

// This implements an init binary for use as pid 1 within a container. Its
// primary task is to launch applications within the container and to handle
// SIGCHLD signals to reap the status for exited processes within the container.
//
// It implements a simple socket based RPC which allows communication with the
// parent process. This is a one way RPC, so it can only be used for basic start
// requests or to check process status.
package stage3

import "C"
