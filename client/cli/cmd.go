// Copyright 2013-2015 Apcera Inc. All rights reserved.

package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	pb "github.com/apcera/kurma/stage1/client"
)

var PanicStack []byte
var PanicError = errors.New("Command panicked!")
var ErrIntResponseQuit = errors.New("User requested Quit at selection")

type cmdDef struct {
	// Name of the command, as invoked on the command line.
	Name string

	// Help text displayed as command line usage.
	Help interface{}

	flagParser flagParser // func allowing the command to interpret flags
	impl       impl       // the command's implementation
	cli        CliWrapper // command line wrapper around impl
}

type Cmd struct {
	// Flags holds any command line args set via flags.
	Flags *flag.FlagSet

	// SubFlags holds any flags passed in after --
	SubFlags    *flag.FlagSet
	RawSubFlags []string

	// ValidatedInput holds a command-specific struct that implements ApcInput.
	// flagParser populates this struct's values from the unflagged args and
	// command line flags.
	ValidatedInput ApcInput

	// Args holds inputs for the named command.
	// In typical use, both Name and Args are set by apc.
	Args []string

	// Client is the Kurma client that uses GrpcConn to communicate with the
	// Kurma server.
	Client pb.KurmaClient

	errChan  chan error // Receives errors returned during execution
	def      *cmdDef    // Command definition
	origArgs []string   // Used for string output
}

// NewCmd inteprets args to match a defined command or alias. The returned *Cmd
// will be nil if args identify a defined command but fail to be parsed by the
// command. Error, when set, will provide details.
func NewCmd(args ...string) (*cmdDef, *Cmd, error) {
	cmd := new(Cmd)
	if err := cmd.parseArgs(args, sanitizeInput(args)); err != nil {
		return cmd.def, nil, err
	}
	return cmd.def, cmd, nil
}

// Makes user input lower case so that apc case does not matter.
func sanitizeInput(args []string) (newArgs []string) {
	newArgs = make([]string, len(args))
	for i := range args {
		newArgs[i] = strings.ToLower(args[i])
	}
	return
}

func (c *Cmd) Name() (name string) {
	if c.IsDefined() {
		name = c.def.Name
	}
	return
}

func (c *Cmd) IsDefined() bool {
	return c.def != nil
}

func (c *Cmd) PrintHelp() {
	if c.IsDefined() {
		c.def.PrintHelp(os.Stderr)
	}
	os.Exit(1)
}

func (c *Cmd) String() string {
	var escapedArgs []string
	for i := range c.origArgs {
		if strings.Contains(c.origArgs[i], " ") {
			escapedArgs = append(escapedArgs, fmt.Sprintf("%q", c.origArgs[i]))
		} else {
			escapedArgs = append(escapedArgs, c.origArgs[i])
		}
	}
	return c.Name() + " " + strings.Join(escapedArgs, " ")
}

// The Go flag parser does not allow arguments and flags to be mixed and expects
// flags to be before the arguments. This function allows argument and flags to
// be mixed.
func (c *Cmd) parseArgs(originalArgs, lowerArgs []string) (err error) {
	c.Flags = flag.NewFlagSet("apc", flag.ContinueOnError)
	// We leave this empty to overwrite the default behavior of Flags.Usage, but do not
	// print help so that we can avoid double-printing command help.
	c.Flags.Usage = func() {}
	addGlobalFlags(c.Flags)

	// Lookup the definition for the command we want to run. Get first non-command index.
	firstArg := 0
	if c.def, firstArg, err = FindCommand(lowerArgs); err != nil {
		return err
	}
	originalArgs = originalArgs[firstArg:]

	// Allow the command to define its flags
	if c.def.flagParser != nil {
		c.def.flagParser(c)
	}
	// Flags are defined at this point so walk the flags and create a map of
	// all allowed flags and not which flags are booleans since they will
	// not have a following argument unless it is of the form -x=true.
	flagMap := map[string]bool{}
	c.Flags.VisitAll(func(f *flag.Flag) {
		boo := false
		if gv, ok := f.Value.(flag.Getter); ok {
			_, boo = gv.Get().(bool)
		}
		flagMap["-"+f.Name] = boo
		flagMap["--"+f.Name] = boo
	})

	// Add aliases for help to flagMap; these should always be defined.
	// NOTE: these must be in our special map, but gostdlib's flag parsing has
	// special cases for help flags that allow us to show help without
	helpFlags := []string{"-h", "--h", "-help", "--help"}
	for _, h := range helpFlags {
		flagMap[h] = true
	}

	// Separate the flags and their values, if any, from the arguments.
	c.Args = []string{}
	var flagArgs []string

	// Tracks if sub flags are present. Sub flags are any flags after --
	// Eg. apc service create servicename -- --ip 5.5.5.5
	subFlags := false

	// Use an index so it can be incremented to skip over flag values more
	// easily.
	for i := 0; i < len(originalArgs); i++ {
		a := originalArgs[i]

		// Ensure the argument has at least one character. If this it isn't, add it
		// to the Args so that the command can handle this oddity. This can happen
		// if you do: apc app create ""
		if len(a) < 1 {
			c.Args = append(c.Args, a)
			continue
		}

		// If we hit "--"" in the args, then break. Anything after this shouldn't be parsed
		// Only stored so we can use it within a specific command.
		// NOTE: this does not support quoted strings as arguments in tests!
		if a == "--" {
			subFlags = true
			continue
		}
		if subFlags {
			c.RawSubFlags = append(c.RawSubFlags, a)
			continue
		}

		// If no - prefix then it is an argument
		if a[0] != '-' {
			c.Args = append(c.Args, a)
			continue
		}
		// Must be a flag. Find out if it has an = embedded.
		eq := strings.IndexByte(a, '=')
		// If it has an = embedded then it is a flag with the value
		// included in the arg.
		if eq > 0 {
			flagArgs = append(flagArgs, a)
			continue
		}
		// Check the flag to see if it is a boolean flag. If it is then
		// just add it to the flag args. If not then a value is expected
		// to grab the next arg as the value.
		boo, found := flagMap[a]
		if !found {
			return fmt.Errorf("%s is not a valid flag", a)
		}
		flagArgs = append(flagArgs, a)
		if !boo {
			i++

			// Check to ensure that we won't go out of bounds here.
			if i < len(originalArgs) {
				flagArgs = append(flagArgs, originalArgs[i])
			}
		}
	}
	// Parse the flags to populate ValidatedInput.
	if err := c.Flags.Parse(flagArgs); err != nil {
		return err
	}
	// Validate the args using the ValidatedInput function.
	if c.ValidatedInput != nil {
		return c.ValidatedInput.ParseArgs(c.Args)
	}
	return nil
}

// RunCli executes the command definition's cli wrapper.
func (c *Cmd) RunCli() error {
	if !c.IsDefined() {
		return fmt.Errorf("Missing command definition! %+v", c)
	}
	return c.def.cli(c)
}

// Run starts the specified command and waits for it to complete.
//
// The returned error is nil if the command runs, has no problems
// with input or output, and returns no error.
func (c *Cmd) Run() error {
	c.Start()
	return c.Wait()
}

func (c *Cmd) Start() {
	c.errChan = make(chan error)
	// Call command, writing any returned error to errChan
	go func() {
		// Add a recovery handler in case anything panics.
		defer func() {
			if r := recover(); r != nil {
				// Dump the stack with information about what command was being run.
				PanicStack = make([]byte, 1<<20) // 1 MB
				runtime.Stack(PanicStack, true)
				header := "Panic during command: " + c.String() + "\n\n"

				// Extend to make room for header.
				PanicStack = append(PanicStack, make([]byte, len(header))...)

				// Copy in the header.
				copy(PanicStack[len(header):], PanicStack)
				copy(PanicStack[:len(header)], header)

				c.errChan <- PanicError
			}
		}()
		c.errChan <- c.def.impl(c)
	}()
}

func (c *Cmd) Wait() error {
	// FIXME: Return an error if not started
	return <-c.errChan
}

// Confirm user input (y/n)
func (cmd *Cmd) ConfirmInput(defaultYes bool, message string) bool {
	if message == "" {
		// Use constant default confirmation.
		message = defaultConfirmation
	}

	var resp string
	if defaultYes {
		resp = askYes
	} else {
		resp = askNo
	}
	for {
		cmd.Ask(message, &resp)

		if resp == askYes || resp == askNo {
			return defaultYes
		}
		resp = strings.ToLower(resp)
		if resp[0] == 'y' {
			return true
		} else if resp[0] == 'n' {
			return false
		}

		// Account for cases where user enters invalid input and we loop.
		if defaultYes {
			resp = askYes
		} else {
			resp = askNo
		}
	}
}

func (cmd *Cmd) ConfirmInputManic(requiredResponse string, message string) bool {
	var resp string

	for {
		cmd.Ask(message, &resp)
		resp = strings.TrimSpace(resp)

		switch resp {
		case requiredResponse:
			return true
		case "":
			continue
		default:
			return false
		}
	}
}

func (cmd *Cmd) Ask(question string, v *string) error {
	fmt.Printf("%s [%s]: ", question, *v)
	var s string
	var err error

	_, err = fmt.Fscanln(os.Stdin, &s)
	if err != nil {
		if err.Error() == "unexpected newline" {
			s = ""
		} else {
			return err
		}
	}

	if s != "" {
		*v = s
	}
	return nil
}

// Like Ask above, but we can give answers with spaces, too.
func (cmd *Cmd) AskWithSpaces(question string, v *string) error {
	fmt.Printf("%s [%s]: ", question, *v)
	reader := bufio.NewReader(os.Stdin)

	s, _ := reader.ReadString('\n')

	if s != "" {
		*v = s
	}

	// Trim trailing newline.
	if strings.HasSuffix(*v, "\n") {
		*v = strings.Split(*v, "\n")[0]
	}
	return nil
}

func (cmd *Cmd) AskInt(question string, v *int) error {
	fmt.Printf("%s [%d]: ", question, *v)
	var s string
	var err error

	_, err = fmt.Fscanln(os.Stdin, &s)
	if err != nil {
		if err.Error() == "unexpected newline" {
			s = ""
		} else {
			return err
		}
	}

	if s == "" {
		return nil
	}

	*v, err = strconv.Atoi(s)
	if err != nil {
		switch strings.ToLower(s) {
		case "q", "quit":
			return ErrIntResponseQuit
		default:
			return err
		}
	}
	return nil
}

// PrintHelp writes the command definition's help string to w with consistent
// whitespace at the beginning and end.
func (c *cmdDef) PrintHelp(w io.Writer) {
	// Don't explode.
	if c.Help == nil {
		fmt.Fprintln(w, "No help found for command:", c.Name)
		return
	}

	var msg string
	switch h := c.Help.(type) {
	case string:
		msg = h
	default:
		msg = "Warning: unable to interpret help for command: " + c.Name
	}

	if msg[0] == '\n' {
		msg = msg[1:]
	}
	fmt.Fprintln(w, msg)
}
