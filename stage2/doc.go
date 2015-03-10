// Copyright 2014-2015 Apcera Inc. All rights reserved.

// +build linux,cgo

// This spawner implementation is used to initiate the creation of a container.
//
// This is implemented looking like a Go binary for ease of use and reusing the
// existing build framework to handle the compilation. The stage2 execution can
// be triggered by setting the SPAWNER_INTERCEPT environment variable. This will
// take over the execution and run the stage2 code.
package stage2

import "C"
