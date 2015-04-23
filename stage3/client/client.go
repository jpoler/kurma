// Copyright 2013-2015 Apcera Inc. All rights reserved.

package client

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// This will manage a connection to a initd daemon.
type Client interface {
	// Tells the initd to chroot into the given directory. If the request is not
	// processed in the given timeout then an error will be returned.
	Chroot(dir string, privileged bool, timeout time.Duration) error

	// SetHostname tells the initd server to set the hostname of the container.
	SetHostname(hostname string, timeout time.Duration) error

	// Tells the initd server to exec the given command/environment and such with
	// the given chroot. This is used to boot a user initd rather than our own.
	Exec(
		command []string, env []string, stdout string, stderr string, timeout time.Duration,
	) error

	// Starts a given named command within the initd server.
	Start(
		name string, command []string, workingDirectory string, env []string,
		stdout string, stderr, user, group string, timeout time.Duration,
	) error

	// Returns the status of all named commands in the container.
	Status(timeout time.Duration) (map[string]string, error)

	// Stops the client, disconnecting all requests and preventing further
	// requests from succeeding.
	Stop()

	// Returns true if this client has been stopped via a call to Stop(). Note
	// that this will start returning true prior to Stop() completing.
	Stopped() bool

	// Waits for any command that has terminated with a sequence number higher
	// than the one given.
	Wait(timeout time.Duration) error

	// Waits for the socket associated with this client to be created. If it does
	// not exist within the timeout window given then this call will return an
	// error.
	WaitForSocket(timeout time.Duration) error
}

// Internal implementation of Client
type client struct {
	// The path to the socket that should be used to issues commands.
	socket string

	// This channel is kept open so long as the client is available.
	stoppedChan chan struct{}

	// A Waitgroup that allows us to ensure that all waiting goroutines have
	// finished before we "close" this client.
	waitGroup sync.WaitGroup

	// Protect Stop() and ONLY Stop()
	stopMutex sync.Mutex
}

// Creates a new client object that uses the given socket file. If the file does
// not exist then this will not throw an error, but future calls to request()
// will.
func New(socketfile string) Client {
	return &client{
		socket:      socketfile,
		stoppedChan: make(chan struct{}),
	}
}

// Performs the given request against the server.
func (c *client) request(request [][]string, timeout time.Duration) ([]string, error) {
	c.waitGroup.Add(1)
	defer c.waitGroup.Done()

	select {
	case <-c.stoppedChan:
		return nil, errors.New("client is shutting down.")
	default:
	}

	makeError := func(err error) ([]string, error) {
		select {
		case <-c.stoppedChan:
			return nil, errors.New("client is stopped.")
		default:
			return nil, err
		}
	}

	// Connect to the socket.
	conn, err := net.Dial("unix", c.socket)
	if err != nil {
		return nil, err
	}

	// Abandon whatever we're doing and close the request if stop is called
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-done:
			// Nothing needs to be done since the request completed first.
			return
		case <-c.stoppedChan:
			// Set the deadline on the connection to 0 in order to force it to
			// disconnect right away. We have to ignore errors here because the
			// connection might have actually been closed but the request channel not
			// signaled yet.
			conn.SetDeadline(time.Now())
		}
	}()

	// Set the timeout on the connection if the user requested one.
	var deadline time.Time

	if timeout > 0 {
		deadline = time.Now().Add(timeout)

		// This is the absolute value for the time that we want this connection to
		// automatically close (and therefore return a read error to the caller).
		if err = conn.SetDeadline(deadline); err != nil {
			return nil, err
		}
	}

	// A simple buffer to make writing strings and such cheaper.
	buffer := bufio.NewWriter(conn)

	// Protocol, always version 1 for now.
	if _, err := buffer.WriteString("1\n"); err != nil {
		return makeError(err)
	}

	// Write the length of the outer string.
	lStr := fmt.Sprintf("%d\n", len(request))
	if _, err := buffer.WriteString(lStr); err != nil {
		return makeError(err)
	}

	// Write the length of each inner string.
	for _, inner := range request {
		lStr := fmt.Sprintf("%d\n", len(inner))
		if _, err := buffer.WriteString(lStr); err != nil {
			return makeError(err)
		}

		// If the length of inner is zero then we need not do any more work.
		if len(inner) == 0 {
			continue
		}

		// Next write the individual strings in the inner array
		for _, s := range inner {
			lStr := fmt.Sprintf("%d\n", len(s))
			// Write the string length.
			if _, err := buffer.WriteString(lStr); err != nil {
				return makeError(err)
			}

			// If the string length is zero then we need not write the actual string.
			if _, err := buffer.WriteString(s); err != nil {
				return makeError(err)
			}
		}
	}

	// Flush the data to the socket.
	if buffer.Flush(); err != nil {
		return makeError(err)
	}

	responseByte, err := ioutil.ReadAll(conn)
	if err != nil {
		return makeError(err)
	}

	// Verify that the results actually have some data in them.
	if len(responseByte) == 0 {
		return makeError(errors.New("Empty response."))
	}

	// Split the results into strings.
	responses := strings.Split(string(responseByte), "\n")

	// Success!
	return responses, nil
}

// Stop stops all current requests and prevents new ones from being processed.
func (c *client) Stop() {
	//Protect against a race where Stop() is called and both threads enter the
	//select{} at the same time.
	c.stopMutex.Lock()
	defer c.stopMutex.Unlock()
	select {
	case <-c.stoppedChan:
		return
	default:
		close(c.stoppedChan)
	}
	c.waitGroup.Wait()
}

// WaitForSocket blocks for timeout, or until the initd socket becomes available.
func (c *client) WaitForSocket(timeout time.Duration) error {
	timeoutExceeded := time.After(timeout)

	// Don't use timer.Tick because we don't want to wait before our initial
	// poll
	for {
		select {
		case <-c.stoppedChan:
			return errors.New("client is being shut down.")
		case <-timeoutExceeded:
			return fmt.Errorf("timeout (%v) waiting for the socket to be created.", timeout)
		default:
			if _, err := os.Stat(c.socket); err == nil {
				return nil
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Implements Client.Stopped()
func (c *client) Stopped() bool {
	select {
	case <-c.stoppedChan:
		return true
	default:
		return false
	}
}

// Returns the socket file that was used to create this client.
func (c *client) SocketFile() string {
	return c.socket
}

// ----------------
// Request Wrappers
// ----------------

// Implements Client.Chroot()
func (c *client) Chroot(dir string, privileged bool, timeout time.Duration) error {
	// Make the request.
	request := [][]string{[]string{"CHROOT", dir, fmt.Sprintf("%v", privileged)}}
	response, err := c.request(request, timeout)
	if err != nil {
		return err
	}

	// We expect two lines, ["REQUEST OK", ""]
	if len(response) != 2 || response[0] != "REQUEST OK" || response[1] != "" {
		return fmt.Errorf("Invalid response: %#v", response)
	}

	// Success!
	return nil
}

// SetHostname sets the hostname within the container.
func (c *client) SetHostname(hostname string, timeout time.Duration) error {
	request := [][]string{[]string{"SETHOSTNAME", hostname}}
	response, err := c.request(request, timeout)
	if err != nil {
		return err
	}

	// We expect two lines, ["REQUEST OK", ""]
	if len(response) != 2 || response[0] != "REQUEST OK" || response[1] != "" {
		return fmt.Errorf("Invalid response: %#v", response)
	}

	// Success!
	return nil
}

// Issues a request to execute a new command.
func (c *client) Exec(
	command []string, env []string, stdout string, stderr string, timeout time.Duration,
) error {
	request := [][]string{
		[]string{"EXEC"},
		command,
		env,
		[]string{stdout, stderr},
	}

	// Make the request.
	response, err := c.request(request, timeout)
	if err != nil {
		return err
	}

	// We expect two lines, ["REQUEST OK", ""]
	if len(response) != 2 || response[0] != "REQUEST OK" || response[1] != "" {
		return fmt.Errorf("Invalid response: %#v", response)
	}

	// Success!
	return nil
}

// Issues a request to start a new command.
func (c *client) Start(
	name string, command []string, workingDirectory string, env []string, stdout string, stderr,
	user, group string, timeout time.Duration,
) error {
	request := [][]string{
		[]string{"START", name},
		command,
		[]string{workingDirectory},
		env,
		[]string{stdout, stderr},
		[]string{user, group},
	}

	// Make the request.
	response, err := c.request(request, timeout)
	if err != nil {
		return err
	}

	// We expect two lines, ["REQUEST OK", ""]
	if len(response) != 2 || response[0] != "REQUEST OK" || response[1] != "" {
		return fmt.Errorf("Invalid response: %#v", response)
	}

	// Success!
	return nil
}

// Get the status of all named processes in the container.
func (c *client) Status(timeout time.Duration) (map[string]string, error) {
	// Make the request.
	request := [][]string{[]string{"STATUS"}}
	response, err := c.request(request, timeout)
	if err != nil {
		return nil, err
	}

	// We expect two lines, ["REQUEST OK", ""]
	if len(response) < 3 || response[0] != "REQUEST OK" {
		return nil, fmt.Errorf("Invalid response: %#v", response)
	}

	// Ensure that length is valid. We expect:
	// REQUEST OK
	//   NAME1
	//   STATUS1
	//   ... repeated
	// END
	// <empty line due to the nature of strings.Split()>
	if (len(response)-3)%2 != 0 {
		return nil, fmt.Errorf("Invalid response: %#v", response)
	}

	// Walk the lines containing name/status pairs.
	results := make(map[string]string, len(response))
	for i := 1; i < len(response)-2; i += 2 {
		results[response[i]] = response[i+1]
	}

	// Success!
	return results, nil
}

// Tells the initd that it needs to chroot into the given directory.
func (c *client) Wait(timeout time.Duration) error {
	// Make the request.
	request := [][]string{[]string{"WAIT"}}
	response, err := c.request(request, timeout)
	if err != nil {
		return err
	}

	// We expect two lines, ["REQUEST OK", ""]
	if len(response) != 2 || response[0] != "REQUEST OK" || response[1] != "" {
		return fmt.Errorf("Invalid response: %#v", response)
	}

	// Success!
	return nil
}
