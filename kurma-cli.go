// Copyright 2014-2015 Apcera Inc. All rights reserved.

// +build ignore cli

package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/apcera/kurma/client/cli"
	"github.com/apcera/util/terminal"

	pb "github.com/apcera/kurma/stage1/client"
	"google.golang.org/grpc"

	_ "github.com/apcera/kurma/client/cli/commands"
)

const (
	ERROR_PREFIX = "Error: "

	defaultKurmaPort = "12311"
)

func main() {
	var exitcode int
	// defer os.Exit first since it would cancel any previous defers.
	defer func() {
		os.Exit(exitcode)
	}()

	// Handle any panic not caught at command-level.
	defer func() {
		if r := recover(); r != nil {
			// Dump the stack with information about what command was being run.
			cli.PanicStack = make([]byte, 1<<20) // 1 MB
			runtime.Stack(cli.PanicStack, true)
			header := "Panic within CLI\n\n"

			// Extend to make room for header.
			cli.PanicStack = append(cli.PanicStack, make([]byte, len(header))...)
			copy(cli.PanicStack[len(header):], cli.PanicStack)
			copy(cli.PanicStack[:len(header)], header)

			reportPanic()
		}
	}()

	handleSignals()

	// Special handling of flags to avoid calling flag.Parse() here. Commands
	// use flag.Flagsets and we don't want to interfere.
	if len(os.Args) == 1 {
		printHelp(os.Stderr)
		exitcode = 1
		return
	} else if len(os.Args) == 2 {
		switch os.Args[1] {
		case "-v", "--v", "-version", "--version":
			if _, ver, err := cli.NewCmd("version"); err == nil {
				ver.RunCli()
			} else {
				fmt.Fprintln(os.Stderr, "version command not defined")
			}
			return
		case "-h", "--h", "-help", "--help":
			printHelp(os.Stdout)
			exitcode = 0
			return
		}
	}

	// Find the command
	cmdDef, cmd, err := cli.NewCmd(os.Args[1:]...)
	if err != nil {
		exitcode = 1
		// Print help and error if command was found but args were incorrect.
		// e.g. any validation errors returned are surfaced here.
		if cmdDef != nil {
			cmdDef.PrintHelp(os.Stderr)
			// Hide any 'help requested' errors from bubbling up, but still show
			// usage errors.
			if err.Error() == "flag: help requested" {
				return
			}
		}
		fmt.Fprintf(os.Stderr, terminal.Colorize(terminal.ColorError, "Error (usage): %s\n"), err)
		exitcode = 1
		return
	}

	// Catch any other '--version' flags.  While instances of "apc --version", etc
	// will be caught above in our len(os.Args) case, we want to take example input
	// like "apc app --version" just in case.
	if cli.ShowVersion {
		if _, ver, err := cli.NewCmd("version"); err == nil {
			ver.RunCli()
		} else {
			fmt.Fprintln(os.Stderr, "version command not defined")
		}
		return
	}

	conn, err := grpc.Dial(net.JoinHostPort(cli.KurmaHost, defaultKurmaPort))
	if err != nil {
		fmt.Fprintf(os.Stderr, terminal.Colorize(terminal.ColorError, ERROR_PREFIX+"%s\n"), err.Error())
		exitcode = 1
		return
	}
	defer conn.Close()
	cmd.Client = pb.NewKurmaClient(conn)

	exitcode = runCommand(cmd)
}

// Returns an exit code for main's deferred os.Exit. Tries to assist the user to
// work around certain types of errors.
func runCommand(cmd *cli.Cmd) int {
	// Run the command and handle the error.
	err := cmd.RunCli()
	switch {
	case err == nil:
		// Successful run.
		return 0
	// case err == utils.CommandAbortedErr:
	// 	// Command aborted by user without error.
	// 	fmt.Fprintln(os.Stdout, "Command aborted by user.\n")
	// 	return 1
	case err == cli.PanicError:
		// Handle panics from commands.
		reportPanic()
		return 1
	}

	return handleTypedErrors(cmd, err)
}

// handleTypedErrors catches typed errors, such as ApiErrors, and prints
// appropriate error messages. It also contains logic necessary for
// reauthenticating and re-executing a command.
func handleTypedErrors(cmd *cli.Cmd, err error) (code int) {
	code = 1
	// Switch off the type of error received, if any.
	switch aerr := err.(type) {
	default:
		// Unknown error received.
		fmt.Fprintf(os.Stderr, terminal.Colorize(terminal.ColorError, ERROR_PREFIX+"%s\n"), aerr.Error())
		fmt.Fprintf(os.Stdout, "Try `kurma-cli help` for more information.\n")
		return
	}
}

// printHelp dispatches into individual defined commands to find their help, as
// needed
func printHelp(stream *os.File) {
	if _, help, err := cli.NewCmd("help"); err == nil {
		help.RunCli()
	} else {
		fmt.Fprintln(os.Stderr, "help command not defined")
	}
}

// handleSignals to deal with Unix signals reaching this process
func handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, os.Kill)
	go func() {
		<-c
		os.Exit(1)
	}()
}

// reportPanic now only displays panics nicely; at some point, should
// auto-collect panics for central logging
func reportPanic() error {
	fmt.Fprintln(os.Stderr, terminal.Colorize(terminal.ColorError, "\nWe encountered an unexpected internal error."))

	// PanicStack is nil, so don't try to write it.
	if cli.PanicStack == nil {
		return nil
	}

	fmt.Fprintln(os.Stderr, string(cli.PanicStack))
	return nil
}
