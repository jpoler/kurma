// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_START_REQUEST_C
#define INITD_SERVER_START_REQUEST_C

#include <errno.h>
#include <fcntl.h>
#include <limits.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <sysexits.h>
#include <unistd.h>

#include <sys/stat.h>
#include <sys/types.h>

#include "cinitd.h"

// Documented in cinitd.h
void start_request(struct request *r)
{
	int i;
	int name_len;
	pid_t pid;
	uid_t uid;
	gid_t gid;
	unsigned long int ul;

	// The expected protocol for an start statement looks like this:
	// {
	//   { "START", ["<NAME>"], },
	//   { "<COMMAND>", ["<ARGS>", ...]},
	//   { "<WORKING DIRECTORY" },
	//   { ["<ENV=VALUE>", ...]},
	//   { "<STDOUTFILE>", "<STDERRFILE>" },
	//   { "<UID>", "<GID>" },
	// }

	INFO("[%d] START request.\n", r->fd);

	// Protocol error conditions.
	if (
			(r->outer_len != 6) ||
			// START/NAME
			(r->data[0][1] != NULL && r->data[0][2] != NULL) ||
			// COMMAND
			(r->data[1][0] == NULL) ||
			// WORKING DIRECTORY
			(r->data[2][1] != NULL) ||
			// ENV (all values are valid.)
			// STDOUTFILE, STDERRFILE
			(r->data[4][0] == NULL) ||
			(r->data[4][1] == NULL) ||
			(r->data[4][2] != NULL) ||
			// UID, GID
			(r->data[5][0] == NULL) ||
			(r->data[5][1] == NULL) ||
			(r->data[5][2] != NULL) ||
			// END
			(r->data[6] != NULL))
	{
		ERROR("[%d] Protocol error.\n", r->fd);
		initd_response_protocol_error(r);
		return;
	}

	// Verify that name is valid if it was provided.
	if (r->data[0][1] != NULL) {
		name_len = strlen(r->data[0][1]);
		if (name_len > MAX_NAME_LENGTH) {
			ERROR("[%d] Name is longer than %d characters: %d\n",
					r->fd, MAX_NAME_LENGTH, name_len);
			initd_response_protocol_error(r);
			return;
		}
		for (i = 0; i < name_len; i++) {
			if (r->data[0][1][i] == '\n') {
				ERROR("[%d] Name contains a \\n at index %d.\n", r->fd, i);
				initd_response_protocol_error(r);
				return;
			}
		}
	} else {
		name_len = 0;
	}

	// Compute the uid and gid
	uid = (uid_t)uidforuser2(r->data[5][0]);
	if (uid < 0) {
		ERROR("[%d] Error in locating UID\n", r->fd);
		initd_response_internal_error(r);
		return;
	}
	gid = (gid_t)gidforgroup2(r->data[5][1]);
	if (gid < 0) {
		ERROR("[%d] Error in locating GID\n", r->fd);
		initd_response_internal_error(r);
		return;
	}

	fflush(NULL);
	pid = fork();
	if (pid == -1) {
		// Error forking.. Bail on this client request.
		ERROR("[%d] Error in fork(): %s\n", r->fd, strerror(errno));
		initd_response_internal_error(r);
		return;
	} else if (pid == 0) {
		// Setup the initial FD's.
		close_all_fds();
		initd_setup_fds(r->data[4][0], r->data[4][1]);

		// Ensure that we are fully root.
		if (setregid(gid, gid) != 0) { _exit(EX_OSERR); }
		if (getgid() != gid) { _exit(EX_OSERR); }
		if (setreuid(uid, uid) != 0) { _exit(EX_OSERR); }
		if (getuid() != uid) { _exit(EX_OSERR); }

		// chdir into the working directory
		if (r->data[2][0] != NULL) {
			if(chdir(r->data[2][0]) == -1) {
				ERROR("[%d] Error setting working directory: %s\n", r->fd, strerror(errno));
				_exit(EX_OSERR);
			}
		}

		if (setenv("PATH", "/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin", 0) == -1) {
			ERROR("[%d] Error setting PATH: %s\n", r->fd, strerror(errno));
			_exit(EX_OSERR);
		}

		if (execvpe(r->data[1][0], r->data[1], r->data[3]) == -1) {
			ERROR("[%d] Error executing \"%s\": %s\n", r->fd, r->data[1][0], strerror(errno));
			_exit(EX_OSERR);
		}

		// This code path should not be reachable, exec must either return an error
		// that we handle or replace the current process.
		ERROR("[%d] Unhandled exec error\n", r->fd);
		_exit(EX_OSERR);
	}

	// Add this process to the list of tracked processes if necessary.
	if (name_len > 0) {
		if (initd_process_new(r->data[0][1], name_len, pid) == NULL) {
			ERROR("[%d] Error in calloc(): %s\n", r->fd, strerror(errno));
			initd_response_internal_error(r);
			return;
		}
	}

	// Success. Inform the caller.
	INFO("[%d] Successful start.\n", r->fd);
	initd_response_request_ok(r);
}

#endif
