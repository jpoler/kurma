// Copyright 2013-2015 Apcera Inc. All rights reserved.

package client

import (
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	tt "github.com/apcera/util/testtool"
)

func TestNew(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)
	tt.TestNotEqual(t, New("some-socket"), nil)
}

func TestClient_Stop(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	// To effectively test stop, we also want to ensure a request is inflight and
	// that it makes that request return.

	socketFile, l := createSocketServer(t)
	defer l.Close()

	setupRequestThatNeverReturns(t, l)

	stopReadyCh := make(chan bool)
	waitDoneCh := make(chan bool)

	client := New(socketFile)

	go func() {
		close(stopReadyCh)
		defer close(waitDoneCh)
		err := client.Wait(time.Minute)
		if err == nil {
			t.Errorf("wait should have returned an error about the client stopping")
			return
		}
		if err.Error() != "client is stopped." {
			t.Errorf("client should have returned an error that the client was stopping, but got: %v", err)
		}
	}()

	select {
	case <-stopReadyCh:
	case <-time.After(5 * time.Second):
		tt.Fatalf(t, "the wait goroutine wasn't setup within 5 seconds")
	}

	tt.TestEqual(t, client.Stopped(), false)
	client.Stop()
	tt.TestEqual(t, client.Stopped(), true)

	select {
	case <-waitDoneCh:
	case <-time.After(5 * time.Second):
		tt.Fatalf(t, "the wait goroutine should have unblocked within 5 seconds")
	}
}

func TestClient_WaitAfterStop(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	socketFile, l := createSocketServer(t)
	defer l.Close()

	setupRequestThatNeverReturns(t, l)

	client := New(socketFile)

	tt.TestEqual(t, client.Stopped(), false)
	client.Stop()
	tt.TestEqual(t, client.Stopped(), true)

	err := client.Wait(time.Minute)
	tt.TestExpectError(t, err)
	tt.TestEqual(t, err.Error(), "client is shutting down.")
}

func TestClient_StopAndWaitRace(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	// This test is going to monkey with the internals. The goal is to get Wait()
	// to start before calling Stop(), but to then have the request flow begin
	// until after Stop() is done. The goal is test a race around the combination
	// of the two.
	//
	// This is done by triggering the Stop() call immediately after
	// l.Accept(). The Wait() call is a request that will never return with a 1
	// minute timeout, and we validate the response from Wait() is about stopping
	// rather than hitting the timeout.

	socketFile, l := createSocketServer(t)
	defer l.Close()

	client := New(socketFile)

	waitDoneCh := make(chan bool)

	go func() {
		l.Accept()
		client.Stop()
	}()

	go func() {
		defer close(waitDoneCh)
		err := client.Wait(5 * time.Second)
		if err == nil {
			t.Errorf("wait should have returned an error about the client stopping")
			return
		}
		if err.Error() != "client is stopped." {
			t.Errorf("client should have returned an error that the client was stopping, but got: %v", err)
		}
	}()

	select {
	case <-waitDoneCh:
	case <-time.After(6 * time.Second):
		tt.Fatalf(t, "the wait goroutine should have unblocked within 5 seconds")
	}
}

func TestClient_WaitForSocket(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	socketFile := tt.TempFile(t)
	tt.TestExpectSuccess(t, os.Remove(socketFile))

	client := New(socketFile)
	err := client.WaitForSocket(10 * time.Millisecond)
	tt.TestExpectError(t, err)

	f, err := os.Create(socketFile)
	tt.TestExpectSuccess(t, err)
	f.Close()

	err = client.WaitForSocket(10 * time.Millisecond)
	tt.TestExpectSuccess(t, err)
}

func TestClient_Chroot(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	socketFile, l := createSocketServer(t)
	defer l.Close()

	var chrootContent string
	readChan := setupReadRequest(t, l, &chrootContent, "REQUEST OK\n")

	client := New(socketFile)
	err := client.Chroot("/chroot", false, time.Second)
	tt.TestExpectSuccess(t, err)

	select {
	case <-readChan:
	case <-time.After(time.Second):
		tt.Fatalf(t, "Expected to have read client response within 1 second")
	}

	expectedRequest := "1\n1\n3\n6\nCHROOT7\n/chroot5\nfalse"
	tt.TestEqual(t, chrootContent, expectedRequest)
}

func TestClient_Exec(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	socketFile, l := createSocketServer(t)
	defer l.Close()

	var execContent string
	readChan := setupReadRequest(t, l, &execContent, "REQUEST OK\n")

	client := New(socketFile)
	err := client.Exec(
		[]string{"/sbin/init", "foo"}, []string{"FOO=bar"}, "/a", "/b", time.Second,
	)
	tt.TestExpectSuccess(t, err)

	select {
	case <-readChan:
	case <-time.After(time.Second):
		tt.Fatalf(t, "Expected to have read client response within 1 second")
	}

	expectedRequest := "1\n4\n1\n4\nEXEC2\n10\n/sbin/init3\nfoo1\n7\nFOO=bar2\n2\n/a2\n/b"
	tt.TestEqual(t, execContent, expectedRequest)
}

func TestClient_Start(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	socketFile, l := createSocketServer(t)
	defer l.Close()

	var startContent string
	readChan := setupReadRequest(t, l, &startContent, "REQUEST OK\n")

	client := New(socketFile)
	err := client.Start(
		"echo", []string{"123"}, "dir", []string{"FOO=bar"},
		"/a", "/b", "123", "456", time.Second,
	)
	tt.TestExpectSuccess(t, err)

	select {
	case <-readChan:
	case <-time.After(time.Second):
		tt.Fatalf(t, "Expected to have read client response within 1 second")
	}

	expectedRequest := "1\n6\n2\n5\nSTART4\necho1\n3\n1231\n3\ndir1\n7\nFOO=bar2\n2\n/a2\n/b2\n3\n1233\n456"
	tt.TestEqual(t, startContent, expectedRequest)
}

func TestClient_Status(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	socketFile, l := createSocketServer(t)
	defer l.Close()

	var statusContent string
	readChan := setupReadRequest(t, l, &statusContent, "REQUEST OK\nfoo\nrunning\nbar\nexit(1)\nend\n")

	client := New(socketFile)
	status, err := client.Status(time.Second)
	tt.TestExpectSuccess(t, err)

	select {
	case <-readChan:
	case <-time.After(time.Second):
		tt.Fatalf(t, "Expected to have read client response within 1 second")
	}

	expectedStatus := map[string]string{"foo": "running", "bar": "exit(1)"}
	tt.TestEqual(t, status, expectedStatus)
}

func TestClient_Wait(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	socketFile, l := createSocketServer(t)
	defer l.Close()

	var waitContent string
	readChan := setupReadRequest(t, l, &waitContent, "REQUEST OK\n")

	client := New(socketFile)
	err := client.Wait(time.Second)
	tt.TestExpectSuccess(t, err)

	select {
	case <-readChan:
	case <-time.After(time.Second):
		tt.Fatalf(t, "Expected to have read client response within 1 second")
	}

	expectedRequest := "1\n1\n1\n4\nWAIT"
	tt.TestEqual(t, waitContent, expectedRequest)
}

func TestClient_WaitTimeout(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	socketFile, l := createSocketServer(t)
	defer l.Close()

	setupRequestThatNeverReturns(t, l)

	client := New(socketFile)
	err := client.Wait(time.Millisecond * 100)
	tt.TestExpectError(t, err)
	tt.TestEqual(t, strings.Contains(err.Error(), "i/o timeout"), true, "Error should have contained i/o timeout", err.Error())
}

// Test that closing a domain socket will allow a call
// to ioutil.ReadAll to return properly.
func TestReadAllOnClose(t *testing.T) {
	tt.StartTest(t)
	defer tt.FinishTest(t)

	sName := "/tmp/cntm_client_test.sock"
	l, err := net.Listen("unix", sName)
	if err != nil {
		t.Fatalf("Error listening to domain socket: %v", err)
	}
	defer os.Remove(sName)

	go func() {
		fd, err := l.Accept()
		if err != nil {
			t.Fatalf("Accept error: %v", err)
		}
		fd.Write([]byte("OK\n"))
		fd.Close()
	}()

	conn, err := net.Dial("unix", sName)
	if err != nil {
		t.Fatalf("Error connecting to domain socket: %v\n", err)
	}
	defer conn.Close()

	if err = conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("Got error setting a read deadline: %v", err)
	}

	_, err = ioutil.ReadAll(conn)
	if err != nil {
		t.Fatalf("Got an error on ioutil.ReadAll: %v\n", err)
	}
}

func createSocketServer(t *testing.T) (string, net.Listener) {
	socketFile := tt.TempFile(t)
	tt.TestExpectSuccess(t, os.Remove(socketFile))

	l, err := net.Listen("unix", socketFile)
	tt.TestExpectSuccess(t, err)

	return socketFile, l
}

func setupReadRequest(t *testing.T, l net.Listener, content *string, response string) <-chan bool {
	doneChan := make(chan bool, 1)
	go func() {
		c, err := l.Accept()
		if err != nil {
			t.Errorf("error in accept: %v", err)
			return
		}
		defer c.Close()

		// Use a channel to ensure the reading is ready before we write the response
		ch := make(chan bool)

		// This literally is a dumb hack. This should be expanded eventually, but
		// the goal was to get tests on the client rather than writing a protocol
		// parser in Go to mirror the one in C. So this sets up a goroutine to read
		// until the connection closes. Can't use a normal ReadAll because there
		// won't be an EOF or anything. Also don't want to write until it can
		// read. This is kind of funky and could be better, but it allows us to
		// tests the raw message as opposed to the parsed payload.
		go func() {
			defer close(doneChan)
			close(ch)
			b, _ := ioutil.ReadAll(c)
			*content = string(b)
		}()

		// Ensure the read goroutine has started then write the request ok response.
		<-ch
		_, err = c.Write([]byte(response))
		if err != nil {
			t.Errorf("error in write: %v", err)
			return
		}
	}()
	return doneChan
}

// This helper basically reads the request, but never writes a response so it
// would never make the client return. This can be used to test timeouts, and
// for Stop behavior.
func setupRequestThatNeverReturns(t *testing.T, l net.Listener) {
	go func() {
		l.Accept()
	}()
}
