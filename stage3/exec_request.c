// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_EXEC_REQUEST_C
#define INITD_SERVER_EXEC_REQUEST_C

#include <errno.h>
#include <fcntl.h>
#include <stdlib.h>
#include <string.h>
#include <sysexits.h>
#include <unistd.h>

#include <sys/stat.h>
#include <sys/types.h>

#include "cinitd.h"

// Documented in cinitd.h
void exec_request(struct request *r)
{
	pid_t pid;

	// The expected protocol for an exec statement looks like this:
	// {
	//	 { "EXEC" },
	//	 { "<COMMAND>", ["<ARGS>", ...]},
	//	 { ["<ENV=VALUE>", ...]},
	//	 { "<STDOUTFILE>", "<STDERRFILE>" }
	// }

	INFO("[%d] EXEC request.\n", r->fd);

	// Protocol error conditions.
	if (
		(r->outer_len != 4) ||
		// EXEC
		(r->data[0][1] != NULL) ||
		// COMMAND
		(r->data[1][0] == NULL) ||
		// ENV (all values are valid.)
		// STDOUTFILE, STDERRFILE
		(r->data[3][0] == NULL) ||
		(r->data[3][1] == NULL) ||
		(r->data[3][2] != NULL) ||
		// END
		(r->data[4] != NULL))
	{
		INFO("[%d] Protocol error.\n", r->fd);
		initd_response_protocol_error(r);
		return;
	}

	fflush(NULL);
	pid = fork();
	if (pid == -1) {
		// Error forking.. Bail on this client request.
		ERROR("[%d] Error in fork(): %s\n", r->fd, strerror(errno));
		initd_response_internal_error(r);
		return;
	} else if (pid != 0) {
		// Exec is funky, we want the parent to actually do the logic, allowing the
		// child to take over where the parent left off.  This allows us to hand off
		// pid 1 to some other process.

		// FIXME: XXX: If any of these commands fail then the container goes
		// crazy.. Do something sane rather than exiting, or ensure that the
		// response can't happen until the exec is complete.

		// Close all fds, stdin, stdout, stderr will be remapped by initd_setup_fds,
		// but they need to be cleared here while remapping stdout, stderr needs to
		// happen after the chroot.
		close_all_fds();

		// Setup the initial FD's.
		initd_setup_fds(r->data[3][0], r->data[3][1]);

		// Ensure that we are fully root.
		if (setregid(0, 0) != 0) { _exit(EX_OSERR); }
		if (getgid() != 0) { _exit(EX_OSERR); }
		if (setreuid(0, 0) != 0) { _exit(EX_OSERR); }
		if (getuid() != 0) { _exit(EX_OSERR); }

		if (setenv("PATH", "/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin", 0) == -1) {
			ERROR("[%d] Error setting PATH: %s\n", r->fd, strerror(errno));
			_exit(EX_OSERR);
		}

		if (execvpe(r->data[1][0], r->data[1], r->data[2]) == -1) {
			ERROR("[%d] Error executing \"%s\": %s\n", r->fd, r->data[1][0], strerror(errno));
			_exit(EX_OSERR);
		}

		// This code path should not be reachable, exec must either return an error
		// that we handle or replace the current process.
		ERROR("[%d] Unhandled exec error\n", r->fd);
		_exit(EX_OSERR);
	}

	// Success. Inform the caller.
	INFO("[%d] Successful EXEC, responding OK.\n", r->fd);
	initd_response_request_ok(r);
}

#endif
