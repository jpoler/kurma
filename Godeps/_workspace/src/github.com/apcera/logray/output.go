// Copyright 2012-2014 Apcera Inc. All rights reserved.

package logray

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Data elements that are passed to an Output's Write function. In order to
// improve performance these are not copied prior to being passed to an Output's
// Write function and as such they should not be modified in the Output.Write
// call.
type LineData struct {
	Fields          map[string]interface{} `json:"fields"`
	Message         string                 `json:"message"`
	Class           LogClass               `json:"class"`
	TimeStamp       time.Time              `json:"time"`
	CallingPackage  string                 `json:"calling_package"`
	CallingFunction string                 `json:"calling_function"`
	SourceFile      string                 `json:"source_file"`
	SourceLine      int                    `json:"source_line"`
}

// Output objects are used as the actual destination for log lines.
type Output interface {
	// Writes a line into this output device.
	Write(data *LineData) error

	// Flushes the buffered output in order to ensure that data is written to this
	// output's destination.
	Flush() error
}

// Function used to create a new output object.
type NewOutputFunc func(u *url.URL) (Output, error)

// This is the type that all outputs get wrapped in. This will preserve the URL
// and other data outside of the object that users are required to modify in
// order to vastly reduce the total API that each implementation of Output will
// need to provide.
type outputWrapper struct {
	// The Output that the output function returned.
	Output Output

	// The URL associated with this output.
	URL *url.URL
}

// Mutex used to control all actions which might cause thread safety issues.
// Typically this Mutex should ONLY ever be used within the functions called by
// the logging goroutines and shouldn't ever be used by anything else.
var updateMutex sync.RWMutex

// Map of url scheme's to NewOutputFunc functions
var newOutputFuncMap map[string]NewOutputFunc

// Maps of url's to Output objects.
var outputMap map[string]*outputWrapper

// Adds a new output type to the map of possible outputs. This defines the
// Scheme that will be provided via the URL. Fragments are managed by the Output
// system rather than being dealt with in this function.
func AddNewOutputFunc(name string, f NewOutputFunc) bool {
	updateMutex.Lock()
	defer updateMutex.Unlock()
	return lockedAddNewOutputFunc(name, f)
}

// Sets up the outputMap with all known URL Scheme parsers.
func lockedSetupOutputMap() {
	newOutputFuncMap = make(map[string]NewOutputFunc, 100)
	newOutputFuncMap["stdout"] = newOutputFuncStdout
	newOutputFuncMap["stderr"] = newOutputFuncStderr
	newOutputFuncMap["file"] = newOutputFuncFile
	newOutputFuncMap["fd"] = newOutputFuncFd
	newOutputFuncMap["discard"] = newOutputFuncDiscard
	outputMap = make(map[string]*outputWrapper, 100)
}

// Inner locked code for AddNewOutputFunc()
func lockedAddNewOutputFunc(name string, f NewOutputFunc) (ok bool) {
	if _, ok := newOutputFuncMap[name]; ok == true {
		return false
	}
	newOutputFuncMap[name] = f
	return true
}

// Creates a new Output destination and configures it as default as
// possible.
func newOutput(uri string) (o *outputWrapper, err error) {
	updateMutex.Lock()
	defer updateMutex.Unlock()
	return lockedNewOutput(uri)
}

// Inner locked code for NewOutput()
func lockedNewOutput(uri string) (o *outputWrapper, err error) {
	o, ok := outputMap[uri]
	if ok == true {
		return o, nil
	}

	// Attempt to parse the URL into something useful.
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	// Get the generation function an call it.
	f, ok := newOutputFuncMap[u.Scheme]
	if ok != true || f == nil {
		return nil, fmt.Errorf("Unknown url scheme: '%s'", u.Scheme)
	}

	// Get the output object for this registered scheme type.
	output, err := f(u)
	if err != nil {
		return nil, err
	} else if output == nil {
		return nil, fmt.Errorf("NewOutputFunc can not return nil, nil")
	}

	// Add the new object back to the map to ensure it gets reused later rather
	// than being constantly recreated.
	wrapper := &outputWrapper{
		Output: output,
		URL:    u,
	}
	outputMap[uri] = wrapper

	return wrapper, nil
}

// ----------------------------
// Default NewOutputFunc functions.
// ----------------------------

// Parses a URL that starts with stdout://
func newOutputFuncStdout(u *url.URL) (Output, error) {
	if u.User != nil {
		return nil, fmt.Errorf("Can not use a username with stdout.")
	}
	if u.Host != "" {
		return nil, fmt.Errorf("Can not use a hostname with stdout.")
	}
	if u.Path != "" {
		return nil, fmt.Errorf("Can not use a path with stdout.")
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("Can not use a fragment with stdout.")
	}

	// Parse the RawQuery so we can extract the format string.
	values, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, err
	}

	// Get the values we need for setting up the Output object.
	format := values.Get("format")
	delete(values, "format")
	color := values.Get("color")
	delete(values, "color")

	// Check that nothing else was defined.
	if len(values) != 0 {
		bad := make([]string, 0, len(values))
		for k, _ := range values {
			bad = append(bad, k)
		}
		return nil, fmt.Errorf(
			"Unknown parameters: %s", strings.Join(bad, ","))
	}

	// The output object we will return.
	return NewIOWriterOutput(os.Stdout, format, color)
}

// Parses a URL that starts with stderr://
func newOutputFuncStderr(u *url.URL) (Output, error) {
	if u.User != nil {
		return nil, fmt.Errorf("Can not use a username with stderr.")
	}
	if u.Host != "" {
		return nil, fmt.Errorf("Can not use a hostname with stderr.")
	}
	if u.Path != "" {
		return nil, fmt.Errorf("Can not use a path with stderr.")
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("Can not use a fragment with stderr.")
	}

	// Parse the RawQuery so we can extract the format string.
	values, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, err
	}

	// Get the values we need for setting up the Output object.
	format := values.Get("format")
	delete(values, "format")
	color := values.Get("color")
	delete(values, "color")

	// Check that nothing else was defined.
	if len(values) != 0 {
		bad := make([]string, 0, len(values))
		for k, _ := range values {
			bad = append(bad, k)
		}
		return nil, fmt.Errorf(
			"Unknown parameters: %s", strings.Join(bad, ","))
	}

	// The output object we will return.
	return NewIOWriterOutput(os.Stderr, format, color)
}

// Parses a URL that starts with file://
func newOutputFuncFile(u *url.URL) (Output, error) {
	if u.User != nil {
		return nil, fmt.Errorf("Can not use a username with file.")
	}
	if u.Host != "" {
		return nil, fmt.Errorf("Can not use a hostname with file.")
	}
	if u.Path == "" {
		return nil, fmt.Errorf("File output must have a path specified.")
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("Can not use a fragment with file.")
	}

	// Parse the RawQuery so we can extract the format string.
	values, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, err
	}

	// Get the values we need for setting up the Output object.
	format := values.Get("format")
	delete(values, "format")
	color := values.Get("color")
	delete(values, "color")

	// Check that nothing else was defined.
	if len(values) != 0 {
		bad := make([]string, 0, len(values))
		for k, _ := range values {
			bad = append(bad, k)
		}
		return nil, fmt.Errorf("Unknown parameters: %s", strings.Join(bad, ","))
	}

	// Open the specified file
	file, err := os.OpenFile(u.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	// The output object we will return.
	return NewIOWriterOutput(file, format, color)
}

// Parses a URL that starts with file://
func newOutputFuncFd(u *url.URL) (Output, error) {
	if u.User != nil {
		return nil, fmt.Errorf("Can not use a username with fd.")
	}
	if u.Path != "" {
		return nil, fmt.Errorf("Can not use a path with fd.")
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("Can not use a fragment with fd.")
	}

	// Parse the RawQuery so we can extract the format string.
	values, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, err
	}

	// Get the values we need for setting up the Output object.
	format := values.Get("format")
	delete(values, "format")
	color := values.Get("color")
	delete(values, "color")

	// Check that nothing else was defined.
	if len(values) != 0 {
		bad := make([]string, 0, len(values))
		for k, _ := range values {
			bad = append(bad, k)
		}
		return nil, fmt.Errorf("Unknown parameters: %s", strings.Join(bad, ","))
	}

	// Parse the fd number from the host
	fd, err := strconv.Atoi(u.Host)
	if err != nil {
		return nil, fmt.Errorf("Invalid host specified, not a valid number.")
	}

	// open the fd
	file := os.NewFile(uintptr(fd), "log")

	// The output object we will return.
	return NewIOWriterOutput(file, format, color)
}

// Parses a URL that starts with discard://
func newOutputFuncDiscard(u *url.URL) (Output, error) {
	return NewIOWriterOutput(ioutil.Discard, "", "")
}
