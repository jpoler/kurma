// Copyright 2012-2014 Apcera Inc. All rights reserved.

package logray

import (
	"fmt"
	"sync"
	"time"
)

// Logger represents the core struct that is used in order to write output from
// the application. It exposes all of the functions for log levels, fields for
// additional metadata, and has the outputs for log data associated with it.
type Logger struct {
	// Fields represents metadata that has been captured to be associated with the
	// log messages.
	Fields map[string]interface{}

	// The outputs configured on the current logger
	outputs     []*loggerOutputWrapper
	outputMutex sync.RWMutex
}

// loggerOutputWrapper is used to match the specific output to the log classes
// that should be sent to it. It is seprate from the outputWrapper, as that is
// intended for re-use of instantiated outputs, where the multiple outputs may
// be configured for different log levels.
type loggerOutputWrapper struct {
	// The configured log class out of the specific output
	Class LogClass

	// The Output the wrapper represents.
	Output Output
}

var (
	// defaultOutputs defines a list of outputs that will be added to a newly
	// created Logger.
	defaultOutputs []*loggerOutputWrapper

	// defaultOutputMutex handles locking around the array of default outputs to
	// add to a new Logger.
	defaultOutputMutex sync.RWMutex
)

// New returns a new Logger with the default configuration.
func New() *Logger {
	defaultOutputMutex.RLock()
	defer defaultOutputMutex.RUnlock()

	logger := &Logger{
		Fields:  make(map[string]interface{}),
		outputs: make([]*loggerOutputWrapper, len(defaultOutputs)),
	}
	copy(logger.outputs, defaultOutputs)

	return logger
}

// AddDefaultOutput adds a new default output which will be used on newly
// created Loggers.
func AddDefaultOutput(uri string, classes ...LogClass) error {
	// generate it
	lo, err := createOutputWrapper(uri, classes)
	if err != nil {
		return err
	}
	if lo == nil {
		return nil
	}

	// Lock the default outputs
	defaultOutputMutex.Lock()
	defer defaultOutputMutex.Unlock()
	defaultOutputs = append(defaultOutputs, lo)
	return nil
}

// ResetDefaultOutput clears all the previously defined default outputs for new
// Loggers.
func ResetDefaultOutput() {
	defaultOutputMutex.Lock()
	defaultOutputs = make([]*loggerOutputWrapper, 0)
	defaultOutputMutex.Unlock()
}

// ResetCachedOutputs clears all the cached outputs that were previously
// instantiated.
func ResetCachedOutputs() {
	updateMutex.Lock()
	defer updateMutex.Unlock()
	lockedSetupOutputMap()
}

// ResetDefaultLogLevel can be used to reconfigure the existing default outputs
// all be at a new log level.
func ResetDefaultLogLevel(classes ...LogClass) {
	// Combine the output classes into just one
	class := NONE
	for _, c := range classes {
		class |= c
	}

	defaultOutputMutex.Lock()
	for _, o := range defaultOutputs {
		o.Class = class
	}
	defaultOutputMutex.Unlock()
}

// Clone returns a new Logger object and copies over the configuration and all
// fields along with it.
func (logger *Logger) Clone() *Logger {
	clone := &Logger{}

	// copy the outputs
	logger.outputMutex.RLock()
	clone.outputs = make([]*loggerOutputWrapper, len(logger.outputs))
	copy(clone.outputs, logger.outputs)
	logger.outputMutex.RUnlock()

	// copy the fields
	clone.Fields = make(map[string]interface{}, len(logger.Fields))
	for k, v := range logger.Fields {
		clone.Fields[k] = v
	}
	return clone
}

// AddOutput adds a new output for the Logger based on the provided URI.
func (logger *Logger) AddOutput(uri string, classes ...LogClass) error {
	// generate it
	lo, err := createOutputWrapper(uri, classes)
	if err != nil {
		return err
	}
	if lo == nil {
		return nil
	}

	// Lock our outputs
	logger.outputMutex.Lock()
	defer logger.outputMutex.Unlock()
	logger.outputs = append(logger.outputs, lo)
	return nil
}

// ResetOutput clears all the previously defined outputs on the Logger.
func (logger *Logger) ResetOutput() {
	logger.outputMutex.Lock()
	logger.outputs = make([]*loggerOutputWrapper, 0)
	logger.outputMutex.Unlock()
}

// createOutputWrapper generates a new loggerOutputWrapper based on the passed
// parameters. It will return an error if it fails to generate the output. It
// will return nil, nil if there are no classes specified.
func createOutputWrapper(uri string, classes []LogClass) (*loggerOutputWrapper, error) {
	// If the caller didn't give us any classes to configure then we do
	// nothing. This can be either an error or not, and since it is just as easy
	// either way we define this as being a no op rather than error condition.
	if len(classes) == 0 {
		return nil, nil
	}

	// Generate the output wrapper or capture the cached one
	ow, err := newOutput(uri)
	if err != nil {
		return nil, err
	}

	// Combine the output classes into just one
	class := NONE
	for _, c := range classes {
		class |= c
	}

	// generate the wrapper
	lo := &loggerOutputWrapper{Class: class, Output: ow.Output}
	return lo, nil
}

// ClearFields resets the Field on the Logger.
func (logger *Logger) ClearFields() {
	logger.Fields = make(map[string]interface{})
}

// RemoveFields will remove any of the mentioned keys from the Logger's Fields.
func (logger *Logger) RemoveFields(keys ...string) {
	for _, s := range keys {
		delete(logger.Fields, s)
	}
}

// SetFields can be used to copy all of the values in the provided fields map to
// the current Logger.
func (logger *Logger) SetFields(fields map[string]interface{}) {
	for k, v := range fields {
		logger.Fields[k] = v
	}
}

// SetField is used to set the specified field to the provided value on the
// current Logger.
func (logger *Logger) SetField(key string, value interface{}) {
	logger.Fields[key] = value
}

// Injects a log in the trace class for this category if logging for this
// category is enabled, otherwise this does nothing. This will format the given
// arguments using fmt.Sprint. If a format string is desired then use Tracef()
// instead.
func (logger *Logger) Trace(args ...interface{}) {
	logger.log(TRACE, fmt.Sprint(args...))
}

// Injects a log in the trace class for this category if logging for this
// category is enabled, otherwise this does nothing. This formats the log line
// using the format string provided (See fmt.Sprintf).
func (logger *Logger) Tracef(format string, args ...interface{}) {
	logger.log(TRACE, fmt.Sprintf(format, args...))
}

// Injects a log in the debug class for this category if logging for this
// category is enabled, otherwise this does nothing. This will format the given
// arguments using fmt.Sprint. If a format string is desired then use Debugf()
// instead.
func (logger *Logger) Debug(args ...interface{}) {
	logger.log(DEBUG, fmt.Sprint(args...))
}

// Injects a log in the debug class for this category if logging for this
// category is enabled, otherwise this does nothing. This formats the log line
// using the format string provided (See fmt.Sprintf).
func (logger *Logger) Debugf(format string, args ...interface{}) {
	logger.log(DEBUG, fmt.Sprintf(format, args...))
}

// Injects a log in the info class for this category if logging for this
// category is enabled, otherwise this does nothing. This will format the given
// arguments using fmt.Sprint. If a format string is desired then use Infof()
// instead.
func (logger *Logger) Info(args ...interface{}) {
	logger.log(INFO, fmt.Sprint(args...))
}

// Injects a log in the info class for this category if logging for this
// category is enabled, otherwise this does nothing. This formats the log line
// using the format string provided (See fmt.Sprintf).
func (logger *Logger) Infof(format string, args ...interface{}) {
	logger.log(INFO, fmt.Sprintf(format, args...))
}

// Injects a log in the warn class for this category if logging for this
// category is enabled, otherwise this does nothing. This will format the given
// arguments using fmt.Sprint. If a format string is desired then use Warnf()
// instead.
func (logger *Logger) Warn(args ...interface{}) {
	logger.log(WARN, fmt.Sprint(args...))
}

// Injects a log in the warn class for this category if logging for this
// category is enabled, otherwise this does nothing. This formats the log line
// using the format string provided (See fmt.Sprintf).
func (logger *Logger) Warnf(format string, args ...interface{}) {
	logger.log(WARN, fmt.Sprintf(format, args...))
}

// Injects a log in the error class for this category if logging for this
// category is enabled, otherwise this does nothing. This will format the given
// arguments using fmt.Sprint. If a format string is desired then use Errorf()
// instead.
func (logger *Logger) Error(args ...interface{}) {
	logger.log(ERROR, fmt.Sprint(args...))
}

// Injects a log in the error class for this category if logging for this
// category is enabled, otherwise this does nothing. This formats the log line
// using the format string provided (See fmt.Sprintf).
func (logger *Logger) Errorf(format string, args ...interface{}) {
	logger.log(ERROR, fmt.Sprintf(format, args...))
}

// log is the internal function which creates the line data for the message and
// pushes it onto the transit channel.
func (logger *Logger) log(logClass LogClass, message string) {
	ld := logger.newLineData(logClass, message)
	b := &backgroundLineLogger{
		lineData: *ld,
		logger:   logger,
	}
	transitChannel <- b
}

// newLineData creates the struct that wraps a log message and will capture the
// source of the logging message from the stack.
func (logger *Logger) newLineData(logClass LogClass, message string) *LineData {
	ld := &LineData{
		Message:   message,
		Class:     logClass,
		TimeStamp: time.Now(),
	}

	ld.Fields = make(map[string]interface{}, len(logger.Fields))
	for k, v := range logger.Fields {
		ld.Fields[k] = v
	}

	packageFilenameLine(ld, 4)
	if logClass == ERROR {
		ld.Fields["stack"] = gatherStack()
	}
	return ld
}
