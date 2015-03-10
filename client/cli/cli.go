// Copyright 2013-2014 Apcera Inc. All rights reserved.

package cli

import (
	"flag"
	"fmt"
	"strings"
)

const (
	defaultConfirmation = "Is this correct?"
	askYes              = "Y/n"
	askNo               = "y/N"
)

var (
	// Verbose increases output in some commands.
	Verbose bool
	// VeryVerbose increases output further.
	VeryVerbose bool
	// ShowVersion triggers the 'version' command when called.
	ShowVersion bool

	// global map of defined commands
	apcCommands = make(map[string]cmdDef)
	// global map of command aliases
	aliasedCommands = make(map[string]cmdDef)
)

// ApcInput describes the minimal functionality required by a Cmd's ValidatedInput.
// Each APC command implements this as needed.
type ApcInput interface {
	ParseArgs([]string) error
	Validate() error
}

// CliWrapper should interpret the command line args into typed input.
type CliWrapper func(*Cmd) error

// the internal implementation of the command
type impl func(*Cmd) error
type flagParser func(*Cmd)

// Register a new command to be available from the command line. The name is the
// name of the command, which may contain spaces for an implied hierarchy, or
// colons for an explicit hierarchy.
//
// Parameters: f inteprets flags from the command line, e is the implementation
// of the command, h is the help message that is presented by 'apc help [cmd]'
//
// We have a choice for h: take interface{} and accept const strings, or switch
// to fmt.Stringer and replace the consts with vars, all wrapped with S()
// (help.S).  For clarity elsewhere, we accept the sacrifice of taking
// interface{} here.  (Since string type doesn't have a String() method)
func DefineCommand(name string, f flagParser, i impl, c CliWrapper, h interface{}) {
	if _, ok := apcCommands[name]; ok {
		fmt.Printf("Command already defined: %q\n", name)
	}
	apcCommands[name] = cmdDef{Name: name, flagParser: f, impl: i, cli: c, Help: h}
}

// DefineAlias allows a defined command to be invoked using an alternate name
func DefineAlias(orig string, alternates ...string) {
	origCmd, ok := apcCommands[orig]
	if !ok {
		panic(fmt.Errorf("DefineAlias: original command not found: %q", orig))
	}
	for _, alt := range alternates {
		aliasedCommands[alt] = origCmd
	}
}

func FindCommand(nameparts []string) (*cmdDef, int, error) {
	// Attempt to match command by name from longest to shortest.
	for i := len(nameparts); i > 0; i-- {
		name := strings.Join(nameparts[0:i], " ")
		if len(apcCommands) == 0 {
			return nil, -1, fmt.Errorf("No commands defined.")
		}
		if def, ok := apcCommands[name]; ok {
			return &def, i, nil
		}
		// Command not found by name, so attempt to lookup by aliases.
		if def, ok := aliasedCommands[name]; ok {
			return &def, i, nil
		}
	}
	return nil, -1, fmt.Errorf("Command not found: %s", strings.Join(nameparts, " "))
}

// addGlobalFlags registers default flags on the FlagSet f.
func addGlobalFlags(f *flag.FlagSet) {
	f.BoolVar(&Verbose, "vv", false, "")
	f.BoolVar(&Verbose, "verbose", false, "")
	f.BoolVar(&VeryVerbose, "vvv", false, "")
	f.BoolVar(&VeryVerbose, "very-verbose", false, "")
	f.BoolVar(&ShowVersion, "version", false, "")
	f.BoolVar(&ShowVersion, "ver", false, "")
	f.BoolVar(&ShowVersion, "v", false, "")
}
