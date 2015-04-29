// Copyright 2013 Apcera Inc. All rights reserved.

package testtool

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// Test an HTTP GET to the given URL. If expectedReturnCode is a value other
// than -1, the test will fail if the response status code doesn't match the
// exptected code. Method returns the string value of the response body and the
// status code.
func TestHttpGet(t Logger, url string, expectedReturnCode int) (string, int) {
	// issue the request
	resp, err := http.Get(url)
	TestExpectSuccess(t, err)
	TestNotEqual(t, resp.Body, nil, "Response body shouldn't ever be nil")

	// read the response
	all, err := ioutil.ReadAll(resp.Body)
	TestExpectSuccess(t, err)
	TestExpectSuccess(t, resp.Body.Close())

	// check if a status code was given and check it if it wasn't -1
	if resp.StatusCode != expectedReturnCode && expectedReturnCode != -1 {
		Fatalf(
			t, "Failed unexpected statuscode for url %s: code=%d, "+
				"expectedCode=%d msg=%s",
			url, resp.StatusCode, expectedReturnCode, string(all))
	}
	return string(all), resp.StatusCode
}

// Test an HTTP POST to the given URL, with the given content type and request
// body.. If expectedReturnCode is a value other than -1, the test will fail if
// the response status code doesn't match the exptected code. Method returns the
// string value of the response body and the status code.
func TestHttpPost(
	t Logger, url string, contentType string, body string, expectedReturnCode int,
) (string, int) {
	// issue the request
	resp, err := http.Post(url, contentType, strings.NewReader(body))
	TestExpectSuccess(t, err)
	TestNotEqual(t, resp.Body, nil, "Response body shouldn't ever be nil")

	// read the response
	all, err := ioutil.ReadAll(resp.Body)
	TestExpectSuccess(t, err)
	TestExpectSuccess(t, resp.Body.Close())

	// check if a status code was given and check it if it wasn't -1
	if resp.StatusCode != expectedReturnCode && expectedReturnCode != -1 {
		Fatalf(
			t, "Failed unexpected statuscode for url %s: code=%d, "+
				"expectedCode=%d msg=%s",
			url, resp.StatusCode, expectedReturnCode, string(all))
	}
	return string(all), resp.StatusCode
}

// Test an HTTP PUT to the given URL, with the given content type and request
// body.. If expectedReturnCode is a value other than -1, the test will fail if
// the response status code doesn't match the exptected code. Method returns the
// string value of the response body and the status code.
func TestHttpPut(
	t Logger, url string, contentType string, body string, expectedReturnCode int,
) (string, int) {
	// create the request
	request, err := http.NewRequest("PUT", url, strings.NewReader(body))
	TestExpectSuccess(t, err)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	// issue the request
	resp, err := http.DefaultClient.Do(request)
	TestExpectSuccess(t, err)
	TestNotEqual(t, resp.Body, nil, "Response body shouldn't ever be nil")

	// read the response
	all, err := ioutil.ReadAll(resp.Body)
	TestExpectSuccess(t, err)
	TestExpectSuccess(t, resp.Body.Close())

	// check if a status code was given and check it if it wasn't -1
	if resp.StatusCode != expectedReturnCode && expectedReturnCode != -1 {
		Fatalf(
			t, "Failed unexpected statuscode for url %s: code=%d, "+
				"expectedCode=%d msg=%s",
			url, resp.StatusCode, expectedReturnCode, string(all))
	}
	return string(all), resp.StatusCode
}

// Test an HTTP GET to the given URL for the amount of time specified in
// duration. This will retry with multiple requests until one is successful or
// it has taken longer than the duration. If expectedReturnCode is a value other
// than -1, the test will fail if the response status code doesn't match the
// exptected code. Method returns the string value of the response body and the
// status code.
func TestHttpGetTimeout(t Logger, url string, expectedReturnCode int, duration time.Duration) (string, int) {
	startTime := time.Now()
	deadline := startTime.Add(duration)

	for ; ; time.Sleep(200 * time.Millisecond) {
		// break out if we're past the deadline
		if time.Now().After(deadline) {
			Fatalf(t, "Unable to receive a valid response from %q within %s", url, duration)
		}

		// issue the request
		resp, err := http.Get(url)
		if err != nil || resp.Body == nil {
			continue
		}

		// read the response
		all, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		resp.Body.Close()

		// check if a status code was given and check it if it wasn't -1
		if (expectedReturnCode == -1 && resp.StatusCode < 400) || (expectedReturnCode == resp.StatusCode) {
			return string(all), resp.StatusCode
		} else {
			continue
		}
	}
}
