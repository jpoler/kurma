// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_WAIT_REQUEST_C
#define INITD_SERVER_WAIT_REQUEST_C

#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sysexits.h>
#include <unistd.h>

#include <sys/stat.h>
#include <sys/types.h>

#include "cinitd.h"

// Documented in cinitd.h
void wait_request(struct request *r)
{
	struct process *p;
	char *data;
	char *data_p;
	int len;

	// The expected protocol for an WAIT statement looks like this:
	// {
	//   { "WAIT" },
	// }
	//
	// This request will not return until a process has exited.

	INFO("[%d] WAIT request.\n", r->fd);

	// Protocol error conditions.
	if ((r->outer_len != 1) || (r->data[0][1] != NULL)) {
		ERROR("[%d] Protocol error.\n", r->fd);
		initd_response_protocol_error(r);
		return;
	}

	// Add this socket to the waiting list.
	if (initd_waiting_socket_add(r) == NULL) {
		// initd_waiting_socket_add responds after it is done waiting. It only
		// returns NULL if it fails to allocate the wait structure. In this case, it
		// will already respond to the request with a protocol error. The
		// initd_request_remove request below is not required then.
		return;
	}

	INFO("[%d] Added to the waiting queue.\n", r->fd);

	// Remove the request object for this since we added a reply above.
	r->fd = 0;
	initd_request_remove(r);
}

#endif
