// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_REQUEST_C
#define INITD_SERVER_REQUEST_C

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
struct request *requests_head;

// Documented in cinitd.h
struct request *initd_request_new(int fd)
{
	struct request *r;
	int flags;

	// Ensure that the file descriptor is non blocking.
	if (initd_setnonblocking(fd)) {
		// initd_setnonblocking() self logs.
		ERROR("[%d] Closing the connection.\n", fd);
		if (initd_close(fd)) {
			// nothing we can do but log and move on.
			ERROR("[%d] error in close(): %s\n", fd, strerror(errno));
		}
		return NULL;
	}

	r = (struct request *) CALLOC(1, sizeof(struct request));
	if (r == NULL) {
		ERROR("[%d] error in calloc(): %s\n", fd, strerror(errno));
		ERROR("[%d] Closing the connection.\n", fd);
		if (initd_close(fd)) {
			// nothing we can do but log and move on.
			ERROR("[%d] error in close(): %s\n", fd, strerror(errno));
		}
		return NULL;
	}

	// Set the startup values.
	r->fd = fd;
	r->protocol = 0;
	r->outer_index = 0;
	r->outer_len = 0;
	r->inner_index = 0;
	r->inner_len = 0;
	r->string_len = 0;
	r->size = 0;
	r->state = PROTO;
	r->buffer = &r->buffer_char;
	r->buffer_len = 1;

	// Add it to the list.
	r->next = requests_head;
	r->prev = NULL;
	requests_head = r;
	if (r->next != NULL) {
		r->next->prev = r;
	}

	return r;
}

// Documented in cinitd.h
void initd_request_remove(struct request *r)
{
	// On close error we basically do nothing since nothing can really be done. We
	// have to just ignore it and move on.
	if (r->fd != 0) {
		ERROR("[%d] Closing the connection.\n", r->fd);
		if (initd_close(r->fd)) {
			ERROR("[%d] error in close(): %s\n", r->fd, strerror(errno));
		}
	}
	r->fd = 0;

	// Remove the requests object from the active list of requests.
	if (r->next != NULL) {
		r->next->prev = r->prev;
	}
	if (r->prev != NULL) {
		r->prev->next = r->next;
	} else {
		requests_head = r->next;
	}

	r->prev = NULL;
	r->next = NULL;

	// Now walk through and free the data structures allocated during this
	// connections request cycle. Currently the only allocated space is in the
	// 'data' element of the structure.
	if (r->data != NULL) {
		char ***p;
		for (p = r->data; *p != NULL; p++) {
			char **q;
			for (q = p[0]; *q != NULL; q++) {
				FREE(*q);
			}
			FREE(*p);
		}
		FREE(r->data);
		r->data = NULL;
	}

	// Free the actual request object.
	FREE(r);
}

// Allocates the given amoune of space, or marks the connection as failed if
// allocating the given space would push the allocation size over the limit for
// this connection.
static void *allocate(struct request *r, int elem, size_t size)
{
	void *p;

	// Safety check the values we were given.
	if (elem < 0) {
		ERROR("[%d] Request is allocating a negative number of elements: %d\n",
				r->fd, elem);
		initd_response_protocol_error(r);
		return NULL;
	} else if (elem > MAX_REQUEST_SIZE) {
		ERROR("[%d] Request is attempting to send too much data: %d\n",
				r->fd, elem);
		initd_response_protocol_error(r);
		return NULL;
	} else if (size < 0) {
		ERROR("[%d] Request is allocating a negative size: %d\n",
				r->fd, (int)size);
		initd_response_protocol_error(r);
		return NULL;
	} else if (size > MAX_REQUEST_SIZE) {
		ERROR("[%d] Request is attempting to send too much data: %u\n",
				r->fd, (unsigned int)size);
		initd_response_protocol_error(r);
		return NULL;
	}

	// See if this would allocate too much memory.
	r->size += (uint64_t)elem * (uint64_t)size;
	if (r->size > MAX_REQUEST_SIZE) {
		ERROR("[%d] Request is over the maximum size of %lu by %lu bytes.\n",
				r->fd, (uint64_t)MAX_REQUEST_SIZE, r->size - MAX_REQUEST_SIZE);
		initd_response_protocol_error(r);
		return NULL;
	}

	// Allocate the memory.
	p = (char *) CALLOC((size_t) elem, size);
	if (p == NULL) {
		ERROR("[%d] Error from calloc(): %s\n", r->fd, strerror(errno));
		initd_response_internal_error(r);
		return NULL;
	}

	return p;
}

// Reads a single character character from the buffer, adds it to the initial
// valid stored in iv and mutates it as though this character was part of the
// initial number. If a return is reached then this will call the function
// pointed to by 'func'.
static int read_int(struct request *r, int *iv, int (*func)(struct request *r))
{
	switch (r->buffer_char) {
	case '0':
	case '1':
	case '2':
	case '3':
	case '4':
	case '5':
	case '6':
	case '7':
	case '8':
	case '9':
		*iv = (*iv * 10) + (int)(r->buffer_char - '0');
		r->buffer = &r->buffer_char;
		r->buffer_len = 1;
		return 0;
	case '\n':
		// End of the string reached, call the function that will process this
		// integer.
		return func(r);
	default:
		ERROR("[%d] Invalid number in length, expected a digit: %d\n",
				r->fd, (int)(r->buffer_char));
		initd_response_protocol_error(r);
		return 1;
	}
}

// Processing after data for a PROTO state has been read.
static int proto(struct request *r)
{
	// Only protocol version 1 is supported for now.
	if (r->protocol != 1) {
		ERROR("[%d] Unknown protocol version: %d\n", r->fd, r->protocol);
		initd_response_protocol_error(r);
		return 1;
	}
	DEBUG("[%d] protocol=%d\n", r->fd, r->protocol);
	DEBUG("[%d] Switching to the OUTER_LEN state.\n", r->fd);
	r->buffer = &r->buffer_char;
	r->buffer_len = 1;
	r->state = OUTER_LEN;
	r->outer_len = 0;
	return 0;
}

// Called when all of the processing on the outer requests has finished.
static int outer_done(struct request *r)
{
	DEBUG("[%d] Request received, processing it.\n", r->fd);

	// The request has finished sending all of its data. Process it.
	r->data[r->outer_index] = NULL;
	r->buffer = &r->buffer_char;
	r->buffer_len = 0;
	r->inner_index = 0;
	r->inner_len = 0;
	r->string_len = 0;

	// Check to see that the user sent a command.
	if (r->data == NULL || r->outer_len < 1 || r->data[0][0] == NULL) {
		ERROR("[%d] Command is missing from request.\n", r->fd);
		initd_response_protocol_error(r);
	} else if (!strncmp(r->data[0][0], "CHROOT", 7)) {
		chroot_request(r);
	} else if (!strncmp(r->data[0][0], "SETHOSTNAME", 12)) {
		sethostname_request(r);
	} else if (!strncmp(r->data[0][0], "EXEC", 5)) {
		exec_request(r);
	} else if (!strncmp(r->data[0][0], "START", 6)) {
		start_request(r);
	} else if (!strncmp(r->data[0][0], "STATUS", 7)) {
		status_request(r);
	} else if (!strncmp(r->data[0][0], "WAIT", 5)) {
		wait_request(r);
	} else {
		// This is an unknown request!
		ERROR("[%d] Unknown command: %s\n", r->fd, r->data[0][0]);
		initd_response_protocol_error(r);
	}

	// Let the reader know that this request is done processing.
	return 1;
}

// Processing after data for a OUTER_LEN state has been read.
static int outer_len(struct request *r)
{
	char ***p;

	// The outer array must be at least 1 element long.
	if (r->outer_len == 0) {
		initd_response_protocol_error(r);
		return 1;
	}

	p = (char ***) allocate(r, r->outer_len + 1, sizeof(char**));
	if (p == NULL) {
		// Allocate internally logs and disconnects the client which makes r no
		// longer valid.
		return 1;
	}
	DEBUG("[%d] outer_len=%d\n", r->fd, r->outer_len);
	DEBUG("[%d] Switching to the INNER_LEN state.\n", r->fd);
	r->data = p;
	r->outer_index = 0;
	r->buffer = &r->buffer_char;
	r->buffer_len = 1;
	r->state = INNER_LEN;
	r->inner_len = 0;
	return 0;
}

// Calls when an inner array has been fully processed.
static int inner_done(struct request *r)
{
	// We encountered the last element in the inner array, move a level higher
	// into the outer array.
	r->data[r->outer_index][r->inner_index] = NULL;
	r->inner_index = 0;
	r->outer_index++;
	if (r->outer_index < r->outer_len) {
		DEBUG("[%d] Switching to the INNER_LEN state.\n", r->fd);
		r->buffer = &r->buffer_char;
		r->buffer_len = 1;
		r->state = INNER_LEN;
		r->inner_len = 0;
		return 0;
	} else {
		return outer_done(r);
	}
}

// Processing after data for a INNER_LEN state has been read.
static int inner_len(struct request *r)
{
	char **p;

	p = (char **) allocate(r, r->inner_len + 1, sizeof(char*));
	if (p == NULL) {
		// Allocate internally logs and disconnects the client which makes r no
		// longer valid.
		return 1;
	}
	DEBUG("[%d] outer_index=%d inner_len=%d\n", r->fd,
			r->outer_index, r->inner_len);

	// These values need set no matter which side of the if statement we walk.
	r->data[r->outer_index] = p;
	r->inner_index = 0;

	// Inner arrays can be 0 elements long but if that happens then we need to
	// possibly skip over this internal array.
	if (r->inner_len == 0) {
		return inner_done(r);
	} else {
		DEBUG("[%d] Switching to the STRING_LEN state.\n", r->fd);
		r->buffer = &r->buffer_char;
		r->buffer_len = 1;
		r->state = STRING_LEN;
		r->string_len = 0;
		return 0;
	}
}

// Called when a string has finished processing.
static int string_done(struct request *r)
{
	// Null the last character in the string for safety.
	*(r->buffer) = '\0';

	DEBUG("[%d] outer_index=%d inner_len=%d string=%s\n", r->fd,
			r->outer_index, r->inner_len,
			r->data[r->outer_index][r->inner_index]);

	// Skip to the next string on the inner_index.
	r->inner_index++;
	if (r->inner_index < r->inner_len) {
		DEBUG("[%d] Switching to the STRING_LEN state.\n", r->fd);
		r->buffer = &r->buffer_char;
		r->buffer_len = 1;
		r->state = STRING_LEN;
		r->string_len = 0;
		return 0;
	}

	// If we got this far then we are done processing all the string on this inner
	// array.
	return inner_done(r);
}

// Processing after data for a STRING_LEN state has been read.
static int string_len(struct request *r)
{
	char *p;

	p = (char *) allocate(r, r->string_len + 1, sizeof(char));
	if (p == NULL) {
		// Allocate internally logs and disconnects the client which makes r no
		// longer valid.
		return 1;
	}
	DEBUG("[%d] outer_index=%d inner_len=%d string_len=%d\n", r->fd,
			r->outer_index, r->inner_len, r->string_len);

	// If the string length we are about to read is zero then we need to
	// transition into the done state right away.
	if (r->string_len == 0) {
		return string_done(r);
	} else {
		DEBUG("[%d] Switching to the STRING state.\n", r->fd);
		r->buffer = r->data[r->outer_index][r->inner_index] = p;
		r->buffer_len = r->string_len;
		r->state = STRING;
		return 0;
	}
}

// Documented in cinitd.h
void initd_request_read(struct request *r)
{
	ssize_t bytes;
	int total = r->buffer_len;

	while (true) {
		// Attempt to read into the string that we are reading from. This can be the
		// size buffer or a real string but either way the call is the same.
		bytes = read(r->fd, r->buffer, r->buffer_len);
		if (bytes == -1) {
			if (errno == EAGAIN || errno == EWOULDBLOCK) {
				// Reading from this file descriptor would have blocked so we return and
				// wait for data to become available later.
				return;
			} else if (errno == EINTR) {
				// We got interrupted by a signal. Retry.
				continue;
			}

			// This is an unknown error.
			ERROR("[%d] Error in read(): %s\n", r->fd, strerror(errno));
			initd_request_remove(r);
			return;
		}
		r->buffer += (int)(bytes);
		r->buffer_len -= (int)(bytes);

		// Check to see if this was the last of the string that needed to be
		// read. If not then we need to bail out since the request is not ready to
		// be processed anyway.
		if (r->buffer_len > 0) {
			DEBUG("Only read %d of %d, looping back\n", (int)bytes, total);
			continue;
		}

		// See what action should be taken based on the state of the request.
		switch (r->state) {
		case PROTO:
			if (read_int(r, &r->protocol, proto)) {
				// read_int logs errors internally.
				return;
			}
			break;
		case OUTER_LEN:
			if (read_int(r, &r->outer_len, outer_len)) {
				// read_int logs errors internally.
				return;
			}
			break;
		case INNER_LEN:
			if (read_int(r, &r->inner_len, inner_len)) {
				// read_int logs errors internally.
				return;
			}
			break;
		case STRING_LEN:
			if (read_int(r, &r->string_len, string_len)) {
				// read_int logs errors internally.
				return;
			}
			break;
		case STRING:
			if (string_done(r)) {
				// string() logs error internally.
				return;
			}
			break;
		default:
			ERROR("[%d] Request is in an unknown state: %d\n", r->fd, r->state);
			initd_response_internal_error(r);
			return;
		}
	}
}

#endif
