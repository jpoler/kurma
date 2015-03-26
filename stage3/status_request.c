// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_STATUS_REQUEST_C
#define INITD_SERVER_STATUS_REQUEST_C

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sysexits.h>
#include <unistd.h>

#include <sys/stat.h>
#include <sys/types.h>

#include "cinitd.h"

// This is the maximum length of the status string that will be written to the
// client. The format is currently one of; running, signaled, exited.  For
// signaled and exited the string will be followed with the signal or exit code,
// which is a number between 0 and 255.  This number is computed using the worst
// case size which will be: signaled(MAX_ASCII_INT) which is 20 + 10.
#define MAXIMUM_STATUS_LENGTH 30

#define END_TOKEN "END\n"

// Write the status to the given data array and return the number of bytes
// written.
int status_snprintf(char *str, size_t size, struct process *p)
{
	if (p->terminated) {
		if (WIFEXITED(p->status)) {
			return snprintf(str, size, "exited(%d)", WEXITSTATUS(p->status));
		} else if (WIFSIGNALED(p->status)) {
			return snprintf(str, size, "signaled(%d)", WTERMSIG(p->status));
		} else {
			return snprintf(str, size, "unknown");
		}
	} else {
		return snprintf(str, size, "running");
	}
}

// Documented in cinitd.h
void status_request(struct request *r)
{
	struct process *p;
	char *data;
	char *data_p;
	int len;

	// The expected protocol for an STATUS statement looks like this:
	// {
	//   { "STATUS" },
	// }
	//
	// The response to a STATUS request includes all named commands in the
	// following format.
	// "REQUEST OK\n"
	// "<NAME>\n<STATUS_STR>\n",
	// ...
	// "END"\n

	INFO("[%d] STATUS request.\n", r->fd);

	// Protocol error conditions.
	if ((r->outer_len != 1) || (r->data[0][1] != NULL)) {
		ERROR("[%d] Protocol error.\n", r->fd);
		initd_response_protocol_error(r);
		return;
	}

	// Start by calculating the size of the start and end token both of these
	// sizes include the \0 at the end of the string we we need to subtract two
	// from the returned value.
	len = sizeof(REQUEST_OK) + sizeof(END_TOKEN) - 2;

	// Loop through each process and allocate enough space to store the result
	// array.
	for (p = process_head; p != NULL; p = p->next) {
		// Add the length of the name string, plus a character for the terminator
		// string
		len += p->name_len + 1;

		// Add the length of an ASCII int64 value and a \n terminator.
		len += MAXIMUM_STATUS_LENGTH + 1;
	}

	// Now Allocate the resulting set.
	data = (char *) CALLOC(len, sizeof(char));
	if (data == NULL) {
		ERROR("[%d] Error in calloc(): %s\n", r->fd, strerror(errno));
		initd_response_internal_error(r);
		return;
	}

	// Copy the REQUEST_OK token (note that sizeof() includes the \0.
	data_p = data;
	memcpy(data_p, REQUEST_OK, sizeof(REQUEST_OK) - 1);
	data_p += sizeof(REQUEST_OK) - 1;

	// Loop through the processes again copying the data.
	for (p = process_head; p != NULL; p = p->next) {
		// Safety check. (2 bytes for \n's).
		if (len - (data_p - data) < p->name_len + MAXIMUM_STATUS_LENGTH + 2) {
			ERROR("[%d] Length of the buffer exceeded!\n", r->fd);
			FREE(data);
			initd_response_internal_error(r);
			return;
		}

		// NAME
		memcpy(data_p, p->name, p->name_len);
		data_p += p->name_len;

		// \n
		(data_p++)[0] = '\n';

		// STATUS_STR
		data_p += status_snprintf(data_p, len - (data_p - data), p);

		// \n
		(data_p++)[0] = '\n';
	}

	// Add the end token. Safety check first to make sure we are not going to
	// overrun our memory. Again we need to remove the '\0' that sizeof() includes
	// in the END_TOKEN length.
	if (data_p + sizeof(END_TOKEN) - 1 > data + len) {
		ERROR("[%d] Length of the buffer exceeded!\n", r->fd);
		FREE(data);
		initd_response_internal_error(r);
		return;
	}
	memcpy(data_p, END_TOKEN, sizeof(END_TOKEN) - 1);
	data_p += sizeof(END_TOKEN) - 1;

	// Respond.
	if (initd_response_add(r->fd, data, (data_p - data), data) == NULL) {
		// initd_response_add logs internally.
		initd_response_internal_error(r);
		return;
	}

	INFO("[%d] Successful status.\n", r->fd);

	// Remove the request object for this since we added a reply above.
	r->fd = 0;
	initd_request_remove(r);
}

#endif
