// Copyright 2014-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SPAWNER_CONTROL_C
#define INITD_SPAWNER_CONTROL_C

#define _GNU_SOURCE

#define FILENAMESIZE 4096

#include <dirent.h>
#include <fcntl.h>
#include <sched.h>
#include <sysexits.h>
#include <unistd.h>

#include "spawner.h"

// Calls dup2() on stdin, stdout, and stderr so they use the file descriptor
// setup in the go side of the this module.
void dup_filedescriptors(int stdinfd, int stdoutfd, int stderrfd) {
	int fd;
	int newfd;

	// Start with stdin using dup2.
	if (stdinfd >= 0)
		if (dup2(stdinfd, 0) == -1) { _exit(EX_OSERR); }

	// Next we do stdout, the following dup will close our existing stdout and
	// replace it with whatever was in the given fd. Note that if stdoutfd is 1
	// then this will be a noop that returns 1.
	if (stdoutfd >= 0)
		if (dup2(stdoutfd, 1) == -1) { _exit(EX_OSERR); }

	// Next we do stderr, the following dup will close our existing stderr and
	// replace it with whatever was in the given fd.
	if (stderrfd >= 0)
		if (dup2(stderrfd, 2) == -1) { _exit(EX_OSERR); }
}

// Walks through all open file descriptors that are not stdin (0), stdout (1),
// or stderr(2) and closes them. On linux this requires us to read the directory
// names in /proc/self/fdinfo. This is fairly lame and ideally will be replaced
// with a call to closefrom() once linux steals that from BSD.
void closefds() {
	DIR *d;
	char buffer[sizeof(struct dirent) + FILENAMESIZE];
	int closed;
	int i;
	struct dirent *results;

	do {
	closed = 0;

	// Open the directory. This shouldn't ever fail since the directory should
	// always exist.
	d = opendir("/proc/self/fdinfo");
	if (d == NULL) { _exit(EX_OSERR); }

	while (1) {
		// Read an element from the directory. This should represent a file
		// descriptor in its name. Errors here are fatal.
		if (readdir_r(d, (struct dirent *) &buffer, &results) != 0) {
		_exit(EX_OSERR);
		}

		// NULL here represents the end of the directory stream.
		if (results == NULL) { break; }

		// Parse the file name into a number.
		i = atoi(results->d_name);

		// If the number is 0, 1, or 2 then we can't close it (thats our stdin,
		// stdout, and stderr), and ifs its the file descriptor of the directory we
		// are iterating then we can't close that either.  In both cases we skip
		// this directory entry.
		if (i < 3 || i == dirfd(d)) {
		continue;
		}

		// Close the file descriptor that we just read.
		if (close(i) == -1) { _exit(EX_OSERR); }
		closed++;
	}

	// Close the directory file descriptor.
	if (closedir(d) == -1) { _exit(EX_OSERR); }

	// Repeat the loop until we can iterate over top of it without closing
	// anything.
	} while (closed != 0);
}

// Joins the various cgroups by writing my pid into the cgroups file.
void joincgroups(char *tasksfiles[]) {
	int fd;
	int i;
	char pidstr[1024];
	int len;

	// Check to ensure we got cgroups and return early if not
	if (tasksfiles == NULL) { return; }

	// Make a string with our pid in it.
	len = sprintf(pidstr, "%d\n", getpid());

	// Join each cgroup one by one.
	for (i = 0; tasksfiles[i] != NULL; i++) {
		fd = open(tasksfiles[i], O_APPEND | O_WRONLY);
	if (fd == -1) { _exit(EX_OSERR); }
	if (write(fd, pidstr, len) != len) { _exit(EX_OSERR); }
	if (close(fd) == -1) { _exit(EX_OSERR); }
	}
}

// Returns the flags that should be used for the outermost clone.
int flags_for_clone(clone_destination_data *args) {
	int flags;

	// If the caller wants a new namespace we set it up here.
	if (args->new_ipc_namespace) { flags |= CLONE_NEWIPC; }
	if (args->new_network_namespace) { flags |= CLONE_NEWNET; }
	if (args->new_mount_namespace) { flags |= CLONE_NEWNS; }
	if (args->new_pid_namespace) { flags |= CLONE_NEWPID; }
	if (args->new_uts_namespace) { flags |= CLONE_NEWUTS; }
	if (args->new_user_namespace) { flags |= CLONE_NEWUSER; }
	return flags;
}

#endif
