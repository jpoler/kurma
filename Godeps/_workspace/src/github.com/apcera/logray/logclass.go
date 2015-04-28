// Copyright 2012-2014 Apcera Inc. All rights reserved.

package logray

import (
	"fmt"
	"strings"
)

// The type used to specify class for log functions.
type LogClass uint32

// Returns string representation of the given class.
func (l LogClass) String() string {
	switch l {
	case NONE:
		return "none"
	case TRACE:
		return "trace"
	case TRACEPLUS:
		return "trace+"
	case DEBUG:
		return "debug"
	case DEBUGPLUS:
		return "debug+"
	case INFO:
		return "info"
	case INFOPLUS:
		return "info+"
	case WARN:
		return "warn"
	case WARNPLUS:
		return "warn+"
	case ERROR:
		return "error"
	case ERRORPLUS:
		return "error+"
	case FATAL:
		return "fatal"
	case FATALPLUS:
		return "fatal+"
	case ALL:
		return "all"
	}

	// The consts do not match the class, so iterate over the base types and
	// generate a combination of the log classes.
	classes := make([]string, 0)
	for _, lc := range baseLogClasses {
		if l&lc == lc {
			classes = append(classes, lc.String())
		}
	}
	return strings.Join(classes, "|")
}

// Parses a logclass string into a LogClass object. This call is case
// insensitive. If the string is not recognised then an error will be returned
// and LogClass will be set to NONE.
func ParseLogClass(s string) (LogClass, error) {
	switch strings.ToLower(s) {
	case "none":
		return NONE, nil
	case "trace":
		return TRACE, nil
	case "trace+":
		return TRACEPLUS, nil
	case "debug":
		return DEBUG, nil
	case "debug+":
		return DEBUGPLUS, nil
	case "info":
		return INFO, nil
	case "info+":
		return INFOPLUS, nil
	case "warn":
		return WARN, nil
	case "warn+":
		return WARNPLUS, nil
	case "error":
		return ERROR, nil
	case "error+":
		return ERRORPLUS, nil
	case "fatal":
		return FATAL, nil
	case "fatal+":
		return FATALPLUS, nil
	case "all":
		return ALL, nil
	}
	return NONE, fmt.Errorf("Invalid LogClass string.")
}

// Returns true if the LogClass object is valid.
func (l LogClass) valid() bool {
	return l&(^(ALL | isPLUSDEF)) == 0
}

// Returns true if this log class includes the TRACE level.
func (l LogClass) includesTrace() bool { return l&TRACE != 0 }

// Returns true if this log class includes the DEBUG level.
func (l LogClass) includesDebug() bool { return l&DEBUG != 0 }

// Returns true if this log class includes the INFO level.
func (l LogClass) includesInfo() bool { return l&INFO != 0 }

// Returns true if this log class includes the WARN level.
func (l LogClass) includesWarn() bool { return l&WARN != 0 }

// Returns true if this log class includes the ERROR level.
func (l LogClass) includesError() bool { return l&ERROR != 0 }

// Returns true if this log class includes the FATAL level.
func (l LogClass) includesFatal() bool { return l&FATAL != 0 }

const (
	// Trace class logging.
	//
	// Logs in this class are typically dumping complete received message, etc. A
	// typical example of Tracing would be to dump the headers and POST data for a
	// request received via HTTP. Generally it should be expected that Trace class
	// calls will only be enabled when specific problems are being tracked.
	TRACE = LogClass(1)

	// Debug class logging.
	//
	// Debugging logging should show the general code path and should be used to
	// highlight edge case code deviations and such. This class will also not be
	// enabled by default.
	DEBUG = LogClass(TRACE << 1)

	// Info class logging.
	//
	// Info class logging is used to inform the admin of situations which are in
	// the normal code path. Examples of Info class logging include startup, shut
	// down notifications, or generalized query logs.
	INFO = LogClass(DEBUG << 1)

	// Warn class logging.
	//
	// Warnings are thrown when a problem happens that can be corrected.  Examples
	// of this type of logging include reconnect or disconnect notices. Logging
	// events in this class should not prevent anything from completing. For
	// example, if a query is delayed, but still completes then this is the right
	// class, but if the query fails then look at Error instead.
	WARN = LogClass(INFO << 1)

	// Error class logging.
	//
	// Error logging is for anything that impacts a query in a way that is
	// unrecoverable. Examples of this might be the inability to reconnect to a
	// backend, or a timeout on an operational call. Generally errors should be
	// reserved for specific failures which indicate that a request is failing.
	ERROR = LogClass(WARN << 1)

	// Fatal class logging.
	//
	// Fatal logs are used to report that something is wrong with the process
	// directly. A completely unexpected nil pointer, or a error writing to the
	// file system, etc. These are things for which recovery may not even be
	// possible and therefor further operation of the process is in question.
	FATAL = LogClass(ERROR << 1)

	// None, used to specify that no logging should be done.
	NONE = LogClass(0)

	// All, used when you want to change settings on all classes at once.
	ALL = (TRACE | DEBUG | INFO | WARN | ERROR | FATAL)

	// A bit that specifically marks the constant as being a plus definition.
	// This is used to ensure that ALL/TRACEPLUS and FATAL/FATALPLUS are different
	// numbers.
	isPLUSDEF = LogClass(FATAL << 1)

	// Trace+ is a set of TRACE, DEBUG, INFO, WARN, ERROR, FATAL and is used in
	// configuration to set all of those classes.
	TRACEPLUS = TRACE | DEBUG | INFO | WARN | ERROR | FATAL | isPLUSDEF

	// Debug+ is a set of DEBUG, INFO, WARN, ERROR, FATAL and is used in
	// configuration to set all of those classes.
	DEBUGPLUS = DEBUG | INFO | WARN | ERROR | FATAL | isPLUSDEF

	// Info+ is a set of INFO, WARN, ERROR, FATAL and is used in configuration to
	// set all of those classes.
	INFOPLUS = INFO | WARN | ERROR | FATAL | isPLUSDEF

	// WARN+ is a set of WARN, ERROR, FATAL and is used in configuration to set
	// all of those classes.
	WARNPLUS = WARN | ERROR | FATAL | isPLUSDEF

	// Trace+ is a set of ERROR, FATAL and is used in configuration to set all of
	// those classes.
	ERRORPLUS = ERROR | FATAL | isPLUSDEF

	// Fatal+ is a set of just FATAL. Its not expected to be used much however it
	// is provided to complete the API.
	FATALPLUS = FATAL | isPLUSDEF
)

var (
	// Defines the base log classes.
	baseLogClasses = []LogClass{
		TRACE, DEBUG, INFO, WARN, ERROR,
	}
)
