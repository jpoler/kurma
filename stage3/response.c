// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_RESPONSE_C
#define INITD_SERVER_RESPONSE_C

#include <errno.h>
#include <fcntl.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <sys/types.h>

#include "cinitd.h"

// This tracks the current list of in flight requests.
struct response *responses_head;

// Documented in cinitd.h.
static struct response *response_add(struct request *req, char *data, int data_len, char *buffer)
{
	struct response *res;

	res = initd_response_add(req->fd, data, data_len, buffer);
	if (res == NULL) {
		// initd_response_add logs internally, and closes the file descriptor for
		// us.
		req->fd = 0;
		initd_request_remove(req);
		return NULL;
	}

	// Now remove the request. Make sure the request's fd doesn't get closed.
	req->fd = 0;
	initd_request_remove(req);

	// Success.
	return res;
}

// Documented in cinitd.h.
struct response *initd_response_add(int fd, char *data, int data_len, char *buffer)
{
	struct response *res;

	// Allocate memory for the structure.
	res = (struct response *) CALLOC(1, sizeof(struct response));
	if (res == NULL) {
		ERROR("[%d] Error in calloc(): %s\n", fd, strerror(errno));
		ERROR("[%d] Closing the connection.\n", fd);
		// There is not much we can do for the connection since all writes require
		// an object. We just close it.
		if (initd_close(fd)) {
			ERROR("[%d] Error in close(): %s\n", fd, strerror(errno));
		}
		return NULL;
	}

	// Set the values on the structure.
	res->fd = fd;
	res->data = data;
	res->data_len = data_len;
	res->buffer = buffer;

	// Add this response to the list.
	res->next = responses_head;
	res->prev = NULL;
	responses_head = res;
	if (res->next != NULL) {
		res->next->prev = res;
	}

	// Success.
	DEBUG("[%d] Initiating response: %s\n", res->fd, data);
	return res;
}

// Documented in cinitd.h.
void initd_response_disconnect(struct response *r)
{
	if (r->fd != 0) {
		INFO("[%d] Closing the connection.\n", r->fd);
		if (initd_close(r->fd)) {
			ERROR("[%d] Error in close(): %s\n", r->fd, strerror(errno));
		}
	}
	r->fd = 0;

    // Remove the requests object from the active list of replies.
	if (r->next != NULL) {
		r->next->prev = r->prev;
	}
	if (r->prev != NULL) {
		r->prev->next = r->next;
	} else {
		responses_head = r->next;
	}
	r->next = NULL;
	r->prev = NULL;

	// Free the buffer if necessary.
	if (r->buffer != NULL) {
		FREE(r->buffer);
		r->buffer = NULL;
	}

	// Free the outer object.
	FREE(r);
}

// Documented in cinitd.h.
void initd_response_write(struct response *r)
{
	ssize_t bytes;

	while (true) {
		if (r->data_len == 0) {
			// Successfully sent the response.
			INFO("[%d] Finished replying.\n", r->fd);
			initd_response_disconnect(r);
			return;
		}

		// Attempts to write another chunk of data to the socket.
		bytes = write(r->fd, r->data, r->data_len);
		if (bytes == -1) {
			if (errno == EAGAIN || errno == EWOULDBLOCK) {
				// Writing data to the socket would have blocked so we return and wait
				// for the buffer space to finish writing the data.
				return;
			} else if (errno == EINTR) {
				// We were interrupted by a signal. Try again.
				continue;
			}

			// This is an unknown error, log it but there is no way to inform the user
			// of the error.
			ERROR("[%d] Error from write(): %s\n", r->fd, strerror(errno));
			initd_response_disconnect(r);
			return;
		}

		// Success, mark the data as read.
		r->data = r->data + (int)(bytes);
		r->data_len -= (int)(bytes);
	}
}

// Documented in cinitd.h.
void initd_response_internal_error(struct request *r)
{
	// Note: sizeof() returns the const length, which includes \0 so we have to
	// subtract one to get the size of just the data.
	response_add(r, INTERNAL_ERROR, sizeof(INTERNAL_ERROR) - 1, NULL);
}

// Documented in cinitd.h.
void initd_response_protocol_error(struct request *r)
{
	// Note: sizeof() returns the const length, which includes \0 so we have to
	// subtract one to get the size of just the data.
	response_add(r, PROTOCOL_ERROR, sizeof(PROTOCOL_ERROR) - 1, NULL);
}

// Documented in cinitd.h.
void initd_response_request_ok(struct request *r)
{
	// Note: sizeof() returns the const length, which includes \0 so we have to
	// subtract one to get the size of just the data.
	response_add(r, REQUEST_OK, sizeof(REQUEST_OK) - 1, NULL);
}

#endif
