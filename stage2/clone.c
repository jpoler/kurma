// Copyright 2014-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SPAWNER_CLONE_C
#define INITD_SPAWNER_CLONE_C

#define _GNU_SOURCE

#define FILENAMESIZE 4096

#include <errno.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include "spawner.h"

void spawn_child(clone_destination_data *args) {
	pid_t child, parent;

	if (args->new_user_namespace) {
		parent = getpid();
		switch (child = fork()) {
		case -1:
			error(1, errno, "fork");
		case 0:
			raise(SIGSTOP);
			writemap(parent, "gid", args->gidmap);
			writemap(parent, "uid", args->uidmap);
			exit(EXIT_SUCCESS);
		}
	}

	setup_container(args, child);
}

static void setup_container(clone_destination_data *args, pid_t uidmap_child) {
	pid_t child;
	int flags;
	int pipe_fd[2];
	char ch;

	// --------------------------------------------------------------------
	// Step 1: Dup the stdoutfd and stderrfd file descriptors into the
	//         stdout and stderr positions.
	// --------------------------------------------------------------------
	DEBUG("Configuring stdin/stdout\n");
	dup_filedescriptors(args->stdinfd, args->stdoutfd, args->stderrfd);

	// --------------------------------------------------------------------
	// Step 2: Close all non 0, 1, 2 file descriptors open in this process.
	// --------------------------------------------------------------------

	// Loop while the call thinks that there are file descriptors that it has not
	// handled yet.
	DEBUG("Closing file descriptors\n");
	closefds();

	// --------------------------------------------------------------------
	// Step 4: Join this process into all cgroups that are listed in the
	//         tasks file section.
	// --------------------------------------------------------------------
	DEBUG("Joining cgroups\n");
	joincgroups(args->tasksfiles);

	// --------------------------------------------------------------------
	// Step 5: Drop privledges to just the current user.
	// --------------------------------------------------------------------
	DEBUG("Resetting uid/gid\n");
	if (setgid(getgid()) < 0 || setuid(getuid()) < 0)
		error(1, 0, "Failed to drop privileges");

	// --------------------------------------------------------------------
	// Step 6: Create the new namespaces.
	// --------------------------------------------------------------------
	flags = flags_for_clone(args);
	if (unshare(flags) < 0)
		error(1, 0, "Failed to unshare namespaces");

	// --------------------------------------------------------------------
	// Step 7: Ensure the uid_map and gid_map files are written.
	// --------------------------------------------------------------------
	if (args->new_user_namespace) {
		DEBUG("Waiting for uidmap/gidmap\n");
		// signal to the side child to write the uid/gid map files
		waitforstop(uidmap_child);
		kill(uidmap_child, SIGCONT);
		waitforexit(uidmap_child);

		// by now, our uid/gid files are written, so escalate to root
		if (setgid(0) < 0 || setgroups(0, NULL) < 0 || setuid(0) < 0)
			error(1, 0, "Failed to get root within the container");
	}

	// --------------------------------------------------------------------
	// Step 8: Setup the root filesystem.
	// --------------------------------------------------------------------
	if (args->container_directory != NULL) {
		DEBUG("Creating root filesystem\n");
		createroot(args->container_directory);
	}

	// --------------------------------------------------------------------
	// Step 9: Prepare for the final fork.
	// --------------------------------------------------------------------

	// Only create the pipe if we're going to detach. The flags are used to
	// coordinate to have the parent not exit until after the filesystem is
	// finished being setup.
	if (args->detach && pipe(pipe_fd) == -1)
		error(1, errno, "pipe");

	// Fork! The namespace changes aren't fully in effect until we fork, such as
	// with a pid namespace, the child will be PID 1, not this process. Also will
	// use this to detach from the namespace if --detach was given. Otherwise, it
	// will wait for it to exit.
	switch (child = fork()) {
	case -1:
		error(1, errno, "fork");
	case 0:
		// create our proc mount and enter the new root
		if (args->new_mount_namespace) {
			DEBUG("Configuring /proc\n");
			mountproc();
		}
		if (args->chroot) {
			DEBUG("Chrooting into filesystem\n");
			enterroot();
		}

		// Signal to the parent that we're ready to exec and we're done with
		// them. This is needed because if the parent exits any sooner, the proc
		// mount consistently fails.
		if (args->detach) {
			DEBUG("Detaching\n");
			close(pipe_fd[1]);
		}

		// --------------------------------------------------------------------
		// Step 10: Remove all existing environment variables.
		// --------------------------------------------------------------------
		environ = NULL;

		// --------------------------------------------------------------------
		// Step 11: Actually perform the exec at this point.
		// --------------------------------------------------------------------
		DEBUG("Exec %s\n", args->command);
		execvpe(args->command, args->args, args->environment);
		error(1, errno, "execvpe");
	}

	// --------------------------------------------------------------------
	// Step 12: End handling for the parent thread.
	// --------------------------------------------------------------------

	// determine if we need to detach or wait for the process to finish
	if (args->detach) {
		// close the write side of the pipe on our end
		close(pipe_fd[1]);

		// Read from the pipe. The child will close their write end when they're
		// done mounting proc and chrooting, that will trigger an EOF waking us up.
		if (read(pipe_fd[0], &ch, 1) != 0) {
			ERROR("Failure in child: read from pipe returned != 0\n");
			exit(1);
		}
	} else {
		// block until the child is done
		waitforexit(child);
	}
}

#endif
