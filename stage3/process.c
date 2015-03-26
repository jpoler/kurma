// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_PROCESS_C
#define INITD_SERVER_PROCESS_C

#include <errno.h>
#include <stdbool.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>

#include <sys/resource.h>
#include <sys/time.h>
#include <sys/types.h>
#include <sys/wait.h>

#include "cinitd.h"

// Documented in cinitd.h.
struct process *process_head;

// Documented in cinitd.h.
struct waiting_socket *waiting_socket_head;

// Documented in cinitd.h.
struct process *initd_process_new(char *name, int name_len, pid_t pid)
{
	struct process *p;

	p = (struct process *) CALLOC(1, sizeof(struct process));
	if (p == NULL) {
		return NULL;
	}

	// Set the member data.
	p->name = (char *) CALLOC(name_len + 1, sizeof(char));
	if (p->name == NULL) {
		FREE(p);
		return NULL;
	}
	memcpy(p->name, name, name_len);
	p->name[name_len] = 0;
	p->name_len = name_len;
	p->pid = pid;

	// Add this process to the list.
	p->next = process_head;
	p->prev = NULL;
	process_head = p;
	if (p->next != NULL) {
		p->next->prev = p;
	}

	// Success.
	return p;
}

// Called on a given process object to notify all the subscribed waiters that
// the process has terminated.
static void notify_waiters(struct process *p)
{
	struct waiting_socket *w;
	struct waiting_socket *wp;
	struct response *r;

	DEBUG("Notifying waiting connections.\n");

	w = waiting_socket_head;
	// Respond to this wait request with an ok reply.
	while (w != NULL) {
		// Walk to the next item and free the current item.
		wp = w;
		w = w->next;

		// Queue up the response. sizeof(REQUEST_OK) return the string size,
		// including the terminating \0 so we need to subtract 1 from the length to
		// only send the data.
		r = initd_response_add(
				wp->fd, REQUEST_OK, sizeof(REQUEST_OK) - 1, NULL);
		if (r == NULL) {
			// Errors are logged internally here.
			initd_waiting_socket_disconnect(wp);
		} else {
			FREE(wp);
		}
	}

	waiting_socket_head = NULL;
	DEBUG("Done notifying waiting connections.\n");
}

// Documented in cinitd.h.
void initd_process_wait(void)
{
	int status;
	struct rusage rusage;
	pid_t pid;
	struct process *p;

	while (true) {
		pid = wait4(-1, &status, WNOHANG, &rusage);
		if (pid == 0 || errno == ECHILD) {
			break;
		} else if (pid == -1) {
			if (errno == EINTR) {
				// The wait() call was interrupted.. This really shouldn't happen but we
				// support it anyway. Allow the loop to repeat.
				continue;
			}

			// Unknown error.
			ERROR("Error in wait4(): %s", strerror(errno));
			return;
		}

		// See if this pid is in the list of pids we care about.
		for (p = process_head; p != NULL; p = p->next) {
			if (p->terminated == false && pid == p->pid) {
				// This is a process that we actually care about.
				DEBUG("proccess '%s' terminated (status=%d)\n",
						p->name, status);
				p->terminated = true;
				p->status = status;
				notify_waiters(p);
				break;
			}
		}

	}
}

// Documented in cinitd.h.
struct waiting_socket *initd_waiting_socket_add(struct request *r)
{
	struct waiting_socket *w;

	// Allocate the waiting structure.
	w = (struct waiting_socket *) CALLOC(1, sizeof(struct waiting_socket));
	if (w == NULL) {
		ERROR("[%d] Error in calloc(): %s\n", r->fd, strerror(errno));
		initd_response_internal_error(r);
		return NULL;
	}

	// Set the values in w.
	w->fd = r->fd;
	w->time_stamp = time(NULL);
	w->next = waiting_socket_head;
	w->prev = NULL;
	waiting_socket_head = w;
	if (w->next != NULL) {
		w->next->prev = w;
	}

	// Success.
	return w;
}

// Documented in cinitd.h.
void initd_waiting_socket_disconnect(struct waiting_socket *w)
{
	ERROR("[%d] Closing connection.\n", w->fd);
	if (initd_close(w->fd)) {
		ERROR("[%d] Error in close(): %s\n", w->fd, strerror(errno));
	}

	if (w->next != NULL) {
		w->next->prev = w->prev;
	}
	if (w->prev != NULL) {
		w->prev->next = w->next;
	} else {
		waiting_socket_head = w->next;
	}
	w->next = NULL;
	w->prev = NULL;
	FREE(w);
}

#endif
