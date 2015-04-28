// Copyright 2012-2014 Apcera Inc. All rights reserved.

package logray

// Interface which defines background workers which can be run.
type backgroundWorker interface {
	Process()
}

// This is the common channel that all log lines will be passed through.
var transitChannel chan backgroundWorker

// This is used to schedule a background flush.
type backgroundFlusher struct {
	logger     *Logger
	updateChan chan bool
}

// Called in order to flush all output's associated with the logger.
func (b *backgroundFlusher) Process() {
	b.logger.outputMutex.RLock()
	defer b.logger.outputMutex.RUnlock()

	for _, o := range b.logger.outputs {
		o.Output.Flush()
	}
	if b.updateChan != nil {
		b.updateChan <- true
	}
}

// This is used to schedule a background log line write.
type backgroundLineLogger struct {
	lineData LineData
	logger   *Logger
}

// Calls the appropriate functions to commit logging data into the proper
// outputs.
func (b *backgroundLineLogger) Process() {
	b.logger.outputMutex.RLock()
	defer b.logger.outputMutex.RUnlock()

	for _, o := range b.logger.outputs {
		if o.Class&b.lineData.Class == b.lineData.Class {
			o.Output.Write(&b.lineData)
		}
	}
}

// Goroutine used to actually perform the logging.
func goLogger() {
	// Note that we currently do not handle or do anything with a panic that is
	// thrown at any point during the log writing process. It is assumed that all
	// writers will manage that internally.  This decision is intentional as
	// recovering from panics might in turn mean that we silently drop logs on the
	// floor.
	for {
		t := <-transitChannel
		t.Process()
	}
}
