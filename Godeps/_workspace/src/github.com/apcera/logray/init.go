// Copyright 2012-2014 Apcera Inc. All rights reserved.

package logray

func init() {
	updateMutex.Lock()
	defer updateMutex.Unlock()

	// Load the default configuration.
	lockedSetupOutputMap()

	// Make sure we have a channel for transiting log data.
	transitChannel = make(chan backgroundWorker, 1000)

	// Setup the flusher goroutine.
	go goLogger()
}
