// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_MAIN_C
#define INITD_SERVER_MAIN_C

#include <ctype.h>
#include <errno.h>
#include <fcntl.h>
#include <string.h>
#include <stdbool.h>
#include <time.h>

#include <sys/socket.h>
#include <sys/time.h>
#include <sys/un.h>

#include "cinitd.h"

// Reads all of the data out of the given file descriptor.
static void clear_pipe(int signal_fd)
{
	char buffer[1024];
	while (true) {
		// Read the contents. Note that we do not preserve the contents at all since
		// they are completely unnecessary.
		if (read(signal_fd, buffer, sizeof(buffer)) == -1) {
			if (errno == EAGAIN || errno == EWOULDBLOCK) {
				return;
			} else if (errno == EINTR) {
				continue;
			}

			ERROR("Error reading from signal fd: %s\n", strerror(errno));
			return;
		}
	}
}

// Prints out the current timestamp.
void server_print_time(FILE* fd)
{
  char buffer[30];
  struct timeval tv;
  time_t curtime;
  struct tm result;

  gettimeofday(&tv, NULL);
  curtime=tv.tv_sec;

  localtime_r(&curtime, &result);
  strftime(buffer,30,"[%T.", &result);
  fprintf(fd, "%s%ld] ", buffer, tv.tv_usec/1000);
}

// Accepts a connection on the listening socket and adds it to the request list.
static void accept_request(int socket_fd)
{
	struct sockaddr_un remote;
	int len;
	int fd;

	len = sizeof(struct sockaddr_un);
	fd = accept(socket_fd, (struct sockaddr *) &remote, &len);
	if (fd == -1) {
		if (errno == EAGAIN || errno == EWOULDBLOCK) {
			// This would have blocked so there is nothing to accept.  bail out.
			return;
		} else if (errno == EINTR) {
			// This operation was interrupted by a signal. Bail out and let the select
			// loop force us back into this function.
			return;
		}
		ERROR("Error in accept(): %s\n", strerror(errno));
		return;
	}

	INFO("[%d] New request received.\n", fd);

	// Set the new file descriptor up as non blocking.
	if (initd_setnonblocking(fd)) {
		// initd_setnonblocking logs internally.
		ERROR("[%d] Closing connection.\n", fd);
		if (initd_close(fd)) {
			ERROR("[%d] Error in close(): %s\n", fd, strerror(errno));
		}
		return;
	}

	if (initd_request_new(fd) == NULL) {
		// initd_request_new logs internally and closes the descriptor on error.
	}
}

// Documented in cinitd.h
void initd_main_loop(int socket_fd, int signal_fd)
{
	struct request *req;
	struct request *req2;
	struct response *resp;
	struct response *resp2;
	struct waiting_socket *ws;
	struct waiting_socket *ws2;
	struct timeval tv;
	fd_set set_reads;
	fd_set set_writes;
	fd_set set_excepts;
	int maxfd;
	int count;
	int select_errno;

	// Log a startup message and then flush both stdout and stderr to make sure
	// that it actually gets written right away.
	INFO("Starting initd.\n");

	while (true) {
		// Zero out the file descriptor sets.
		FD_ZERO(&set_reads);
		FD_ZERO(&set_writes);
		FD_ZERO(&set_excepts);
		count = 0;

		// Add the accepting socket file descriptor.
		maxfd = socket_fd;
		FD_SET(socket_fd, &set_reads);
		count++;

		// Add the signal notification file descriptor.
		if (signal_fd > maxfd) { maxfd = signal_fd; }
		FD_SET(signal_fd, &set_reads);
		count++;

		// Loop through all of the forms of watched sockets adding them to the set.
		for (req = requests_head; req != NULL; req = req->next) {
			count++;
			FD_SET(req->fd, &set_reads);
			FD_SET(req->fd, &set_excepts);
			if (req->fd > maxfd) { maxfd = req->fd; }
		}
		for (resp = responses_head; resp != NULL; resp = resp->next) {
			count++;
			FD_SET(resp->fd, &set_writes);
			FD_SET(resp->fd, &set_excepts);
			if (resp->fd > maxfd) { maxfd = resp->fd; }
		}
		for (ws = waiting_socket_head; ws != NULL; ws = ws->next) {
			count++;
			FD_SET(ws->fd, &set_excepts);
			if (ws->fd > maxfd) { maxfd = ws->fd; }
		}

		// Timeout every minute or so.
		tv.tv_sec = 60;
		tv.tv_usec = 0;

		DEBUG("Entering select on %d file descriptors.\n", count);

		// Select.
		count = select(maxfd + 1, &set_reads, &set_writes, &set_excepts, &tv);

		// Check for errors in the signal handler loop and log them. Since this
		// block can impact the errno that select returned its imparitive that we
		// preserve and restore it.
		if (signal_handler_errno != 0) {
			select_errno = errno;
			ERROR(
					"The signal handler had an error writing to "
					"signal pipe (%s) Zombies may not have been reaped "
					"until now.\n",
					strerror(signal_handler_errno));
			signal_handler_errno = 0;

			// Since there was an error in the signal handler its not clear if the
			// signal fd was written too. Its safest to just clear the pipe and wait
			// for children.
			clear_pipe(signal_fd);
			initd_process_wait();
			errno = select_errno;
		}

		// Check the results of the select.
		if (count == -1) {
			if (errno == EINTR) {
				// Non error, select was interrupted by a signal which was likely
				// SIGCHLD. Reap child processes.
				DEBUG("Select was interrupted.\n");
				initd_process_wait();
				continue;
			}

			ERROR("Error in select(): %s\n", strerror(errno));
			continue;
		} else if (count == 0) {
			// Timeout.. We attempt to reap child processes here as well.
			DEBUG("Select timed out.\n");
			initd_process_wait();
			continue;
		}
		DEBUG("Select triggered on %d file descriptors.\n", count);

		// If the socket file descriptor has triggered then we need to accept on it.
		if (FD_ISSET(socket_fd, &set_reads)) {
			accept_request(socket_fd);
		}

		// If the Signal file descriptor has triggered then we need to read all the
		// data out of it so that we do not fill the buffer by accident.
		if (FD_ISSET(signal_fd, &set_reads)) {
			// Clear the signal pipe, then wait for processes. Order here matters we
			// we need to ensure that a signal received during the initd_process_wait
			// call causes the pipe to become ready to read again.
			clear_pipe(signal_fd);
			initd_process_wait();
		}

		// Walk the list of requesting file descriptors. Note that we need to be
		// safe while walking this list since calls to the various functions we are
		// calling may actually free the req object.
		for (req = requests_head; req != NULL; ) {
			// Pre walk req to the next element before any future call has a chance to
			// free it.
			req2 = req;
			req = req->next;

			if (FD_ISSET(req2->fd, &set_excepts)) {
				// There was an exception on this file descriptor. Close it.
				FD_CLR(req2->fd, &set_excepts);
				initd_request_remove(req2);
			} else if (FD_ISSET(req2->fd, &set_reads)) {
				// We can read more data from the buffer.
				FD_CLR(req2->fd, &set_reads);
				initd_request_read(req2);
			}
		}

		// Now walk the list of responding file descriptors. Note that just like
		// above we need to store a copy since the various calls we are making may
		// actually mutate this list as we are reading it.
		for (resp = responses_head; resp != NULL; ) {
			// Pre walk resp to the next element and use resp2 from here on out.
			resp2 = resp;
			resp = resp->next;

			if (FD_ISSET(resp2->fd, &set_excepts)) {
				// There was an error with this descriptor.. close it.
				FD_CLR(resp2->fd, &set_excepts);
				initd_response_disconnect(resp2);
			} else if (FD_ISSET(resp2->fd, &set_writes)) {
				// We can write more data into the buffer.
				FD_CLR(resp2->fd, &set_writes);
				initd_response_write(resp2);
			}
		}

		// Now walk through all of the waiting sockets. Again we need safe iteration
		// of the list since the disconnect call frees the object which makes
		// reading next dangerous.
		for (ws = waiting_socket_head; ws != NULL; ) {
			// Pre walk ws to the next element.
			ws2 = ws;
			ws = ws->next;

			if (FD_ISSET(ws2->fd, &set_excepts)) {
				// An exception happened.. Disconnect the socket.
				FD_CLR(ws2->fd, &set_excepts);
				initd_waiting_socket_disconnect(ws2);
			}
		}

	}
}


#endif
