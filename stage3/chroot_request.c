// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_CHROOT_REQUEST_C
#define INITD_SERVER_CHROOT_REQUEST_C

#include <errno.h>
#include <string.h>

#include "cinitd.h"

// Documented in cinitd.h
void chroot_request(struct request *r)
{
	bool privileged;

	// The expected protocol for a chroot statement looks like this:
	// {
	//   { "CHROOT" "DIRECTORY" "PRIVILEGED" },
	// }

	INFO("[%d] CHROOT request.\n", r->fd);

	// Protocol error conditions.
	if (
		(r->outer_len != 1) ||
		// CHROOT
		(r->data[0][1] == NULL) ||
		(r->data[0][2] == NULL) ||
		(r->data[0][3] != NULL) ||
		// END
		(r->data[1] != NULL))
	{
		INFO("[%d] Protocol error.\n", r->fd);
		initd_response_protocol_error(r);
		return;
	}

	// Check the privileged flag
	privileged = (strncmp(r->data[0][2], "true", 5) == 0);

	// Attempt the actual chroot.
	if (pivot_root(r->data[0][1], privileged) != 0) {
		ERROR("[%d] Failed to pivot_root('%s'): %s\n", r->fd, r->data[0][1], strerror(errno));
		initd_response_internal_error(r);
		return;
	}

	// Success. Inform the caller.
	INFO("[%d] Successful pivot_root('%s') and chdir('/'), responding OK.\n", r->fd, r->data[0][1]);
	initd_response_request_ok(r);
}

#endif
