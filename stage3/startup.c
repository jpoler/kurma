// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_STARTUP_C
#define INITD_SERVER_STARTUP_C

#include <errno.h>
#include <limits.h>
#include <signal.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include <sys/resource.h>
#include <sys/socket.h>
#include <sys/time.h>
#include <sys/types.h>
#include <sys/un.h>
#include <sys/wait.h>

#include "cinitd.h"

// This is defined in the linux internals and not exposed easily. We have to
// redefine it here.
#define UNIX_PATH_MAX 108

// This is a simple function that opens the socket file that is defined in the
// INITD_SOCKET environment variable.
static int open_socket_file()
{
	char socket_file[PATH_MAX + 1];
	size_t socket_file_len;
	int fd;
	struct sockaddr_un address;
	socklen_t address_length;

	// Copy the location of the socket file into a local buffer.
	memset(socket_file, 0, sizeof(socket_file));
	strncpy(socket_file, getenv("INITD_SOCKET"), PATH_MAX);
	socket_file_len = strlen(socket_file);

	// Socket creation is stupid and has a very low length for path names.  Double
	// check here that the path is within that length.
	if (socket_file_len + 1 > UNIX_PATH_MAX) {
		ERROR("socket file (%s) is too long (%d is longer than %d)\n",
				socket_file, (int)socket_file_len, (int)UNIX_PATH_MAX);
		exit(1);
	}

	// Create a file descriptor for the socket.
	fd = socket(PF_UNIX, SOCK_STREAM, 0);
	if (fd < 0) {
		fprintf(stderr, "Error in socket(): %s\n", strerror(errno));
		exit(1);
	}

	// Ensure that address is zeroed.
	memset(&address, 0, sizeof(struct sockaddr_un));

	// Setup the address structure.
	address.sun_family = AF_UNIX;
	memcpy(address.sun_path, socket_file, sizeof(address.sun_path));

	// Bind the socket.
	if (bind(fd, (struct sockaddr *) &address, sizeof(struct sockaddr_un))) {
          ERROR("socket file (%d) can not be bound, exiting!\n", fd);
		fprintf(stderr, "Error in bind(): %s\n", strerror(errno));
		exit(1);
	}

	// Listen to the socket.
	if (listen(fd, MAX_REQUEST_BACKLOG)) {
          ERROR("socket file (%d) can not be listened on, exiting!\n", fd);
		fprintf(stderr, "Error in listen(): %s\n", strerror(errno));
		exit(1);
	}

	// Set the file descriptor up as non blocking.
	if (initd_setnonblocking(fd)) {
          ERROR("socket file (%d) can not be made non-blocking, exiting!\n", fd);
		fprintf(stderr, "Error marking the socket non blocking: %s\n",
				strerror(errno));
		exit(1);
	}

	DEBUG("Socket file %s opened.\n", socket_file);
	return fd;
}

// This is the file descriptor that the signal handle will write too.
static int signal_handler_fd;

// Documented in cinitd.h.
volatile sig_atomic_t signal_handler_errno;

// This function is called when a SIGCHLD signal is received.
static void signal_sigchld(int sig)
{
	char data;
	data = 0;

	// Attempt to write to the signal_handler_fd descriptor.
	while (true) {
		signal_handler_errno = 0;

		if (write(signal_handler_fd, &data, 1) == -1) {
			if (errno == EAGAIN || errno == EWOULDBLOCK) {
				// This would have been a blocking write, which means that the buffer is
				// full. If this is the case then the select will trigger so this is all
				// moot anyway.
				return;
			} else if (errno == EINTR) {
				// Try again.
				continue;
			}

			// Record this error for select to report later.
			signal_handler_errno = errno;
		}
		return;
	}
}

// This function will install signal_sigchld (above) as the signal handler for
// SIGCHLD signals.
static int setup_signal_handler(void)
{
	struct sigaction sigchld_handler;
	int pipes[2];

	// Clear out any random values left in the error tracking variable before we
	// start.
	signal_handler_errno = 0;

	// Make a pipe that the signal handler will use to interrupt the select loop
	// reliably.
	if (pipe(pipes)) {
		fprintf(stderr, "Error in pipe(): %s\n", strerror(errno));
		exit(1);
	}

	// Mark both pipes non blocking.
	if (initd_setnonblocking(pipes[0]) || initd_setnonblocking(pipes[1])) {
		fprintf(stderr, "Error making pipe non blocking: %s\n",
				strerror(errno));
		exit(1);
	}

	// Set the file descriptor to the writer side of the pipe.
	signal_handler_fd = pipes[1];

	// Zero out the structure.
	memset(&sigchld_handler, 0, sizeof(struct sigaction));

	// Set the required values.
	sigchld_handler.sa_handler = signal_sigchld;
	sigchld_handler.sa_flags = 0;
	if (sigemptyset(&sigchld_handler.sa_mask)) {
		fprintf(stderr, "Error in sigemptyset(): %s\n", strerror(errno));
		exit(1);
	}
	if (sigaction(SIGCHLD, &sigchld_handler, NULL)) {
		fprintf(stderr, "Error in sigaction(): %s\n", strerror(errno));
		exit(1);
	}

	// Return the reader side of the pipe.
	DEBUG("Setup signal handler.\n");
	return pipes[0];
}

// This is a basic C wrapper that is used to intercept the running process if
// its PID 1, which can only happen if this is the initd for a new
// container. This code _MUST_ execute before the golang source starts.  In
// order to do this we use a construct known to work in GCC and clang which
// allows a specific set of code to execute before main(). This is a C
// "constructor".
//
// With this constructor we can ensure that we run our cinitd() function prior
// to golang startup for cgroups_initd.
void cinitd(int argc, char **argv) __attribute__ ((constructor));

// This function is executed prior to golang's startup logic.
void cinitd(int argc, char **argv)
{
	int socket_fd;
	int signal_fd;

	// Ensure we're intended to run
	if (getenv("INITD_INTERCEPT") == NULL) {
		return;
	}

	// Enable debugging if necessary.
	if (getenv("INITD_DEBUG") != NULL) {
		cinitd_debugging = true;
		DEBUG("Debugging logs enabled.\n");
	}

	// Reset our cmdline to just be "init". This doesn't hide the binary path on
	// the host, since that can be seen by /proc/1/exe, but it at least looks
	// slightly better in ps.
	strncpy(argv[0], "init", strlen(argv[0]));

	// It is now safe to assume that we are a naked C program imitating a real
	// initd. Start by receiving our configuration via stdin.

	// Start off by opening the socket file that we use for communication back to
	// the mother ship.
	socket_fd = open_socket_file();

	// Next setup the signal handler that is used to watch for SIGCHLD events.
	signal_fd = setup_signal_handler();

	// And now kick off the inid main loop.
	initd_main_loop(socket_fd, signal_fd);

	// Ensure that we never ever fall back into the go world.
	exit(1);
}

#endif
