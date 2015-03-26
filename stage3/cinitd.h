// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_INITD_H
#define INITD_SERVER_INITD_H

#include <signal.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <unistd.h>

#include <sys/types.h>

// The maximum number of requests that can exist in the listen queue for the
// socket.
#define MAX_REQUEST_BACKLOG 10

// The maximum size any request can allocate. (This includes arrays)
#define MAX_REQUEST_SIZE 1024 * 1024

// The maximum length of a name of a process. This is defined in
// common/constants as well.
#define MAX_NAME_LENGTH 100

// --------------
// Signal Handler
// --------------

// This is a error tracking value used to let the main function report errors
// for the signal handler since we do not want to do IO from inside of it.
extern volatile sig_atomic_t signal_handler_errno;

// -----------------------
// Static Response Strings
// -----------------------

#define INTERNAL_ERROR "INTERNAL ERROR\n"
#define PROTOCOL_ERROR "PROTOCOL ERROR\n"
#define REQUEST_OK "REQUEST OK\n"
#define WAIT_TIMEOUT "WAIT TIMEOUT\n"

// -----------------
// Request handlers.
// -----------------

// This structure tracks the state of a given request. Requests are where the
// client is still writing data on the socket that has not been fully processed.
struct request {
	// The file descriptor representing this request.
	int fd;

	// A temporary buffer character used when reading string integers off the
	// socket one character at a time.
	char buffer_char;

	// This is the buffer that the async read call will use, as well as the total
	// length of the buffer that can be read.
	char *buffer;
	int buffer_len;

	// The protocol that this request is using. For now this is ALWAYS 1.
	int protocol;

	// This is the data being read from the client. It represents a 3 layer byte
	// array which is laid out in the format described above.
	// ELEMENT -> [DATA1, DATA2, DATA3]
	char ***data;

	// This manages that state about which object is being read next.
	enum state {PROTO, OUTER_LEN, INNER_LEN, STRING_LEN, STRING} state;

	// This is the index, and length into the outer array that is currently being
	// read. If either of these are -1 then the length of the outer array has not
	// been read.
	int outer_index;
	int outer_len;

	// This is the index and length into the inner array that is currently being
	// read. If either of these are -1 then the length of the inner array has not
	// been read.
	int inner_index;
	int inner_len;

	// This is the length of the inner most string being read.
	int string_len;

	// The total allocation size for this request.
	uint64_t size;

	// The linked list elements that keep track of the request list.
	struct request *next;
	struct request *prev;
};

// The list of in flight requests. This is a standard list which is terminated
// when next == NULL.
extern struct request *requests_head;

// This creates a new request object based on the given file descriptor and adds
// it to the list of in flight requests. This will return NULL on error.
struct request *initd_request_new(int fd);

// This removes the given request object. If r->fd is non 0 then this will also
// close the file descriptor in the process.
void initd_request_remove(struct request *r);

// This will process a read request from the given request object. This is non
// blocking.
void initd_request_read(struct request *r);

// This is called once a request object is found that has a COMMAND element set
// to "CHROOT".
void initd_chroot_request(struct request *r);

// This is called once a request object is found that has a COMMAND element set
// to "SETHOSTNAME".
void initd_sethostname_request(struct request *r);

// This is called once a request object is found that has a COMMAND element set
// to "EXEC".
void initd_exec_request(struct request *r);

// This is called once a request object is found that has a COMMAND element set
// to "START".
void initd_start_request(struct request *r);

// This is called once a request object is found that has a COMMAND element set
// to "STATUS".
void initd_status_request(struct request *r);

// This is called once a request object is found that has a COMMAND element set
// to "WAIT".
void initd_wait_request(struct request *r);

// ------------------
// Response handlers.
// ------------------

// This structure tracks the state of a given response. Responses are where the
// server (us) is still writing data to the socket prior to the socket being
// closed.
struct response {
	// The file descriptor being worked on.
	int fd;

	// The buffer allocated for the response. If this is NULL then no buffer was
	// allocated and therefor the response requires no free beyond the response
	// structure directly.
	char *buffer;

	// A pointer to the char* array that is being written to the caller.
	char *data;

	// The length of the remaining data in data.
	int data_len;

	// Pointers to the next and previous elements in the list.
	struct response *next;
	struct response *prev;
};

// The list of in flight responses. This is a standard doubly linked list that
// is terminated when next == NULL.
extern struct response *responses_head;

// Adds an asynchronous response on the given file descriptor to the response
// list. The arguments data and data len specify the data to be send, and
// buffer, if non NULL will be freed when the response completes.
struct response *initd_response_add(
		int fd, char *data, int data_len, char *buffer);

// Called when a given response object has become available for writing.
void initd_response_write(struct response *r);

// Disconnects the given response and frees all the memory used by it.
void initd_response_disconnect(struct response *r);

// A quick wrapper for responding with the internal error message to the
// caller. This will free the passed in request object.
void initd_response_internal_error(struct request *r);

// A quick wrapper for responding with the protocol error message to the
// caller. This will free the passed in request object.
void initd_response_protocol_error(struct request *r);

// -----------------
// Process handlers.
// -----------------

// A linked list of all file descriptors that are waiting on a process to exit.
struct waiting_socket {
	// The file descriptor.
	int fd;

	// The time that this connection was created.
	time_t time_stamp;

	// The next and prev pointers for structures in this list.
	struct waiting_socket *next;
	struct waiting_socket *prev;
};

// This structure tracks a running process started via the start command.
struct process {
	// The name given to this process.
	char *name;

	// The length of the name string.
	int name_len;

	// The pid of the process started.
	pid_t pid;

	// This is set to true if the process has terminated in some way, otherwise
	// this is zero.
	bool terminated;

	// The status of the process if it has exited.
	int status;

	// A pointer to the next and previous elements of the list.
	struct process *next;
	struct process *prev;
};

// A list of all processes running within this initd. This is next terminated.
extern struct process *process_head;

// Creates a new process object with the given name, name_len, and pid.
struct process *initd_process_new(char *name, int name_len, pid_t pid);

// Called when a child may have exited. This will spawn notifications to all
// waiting clients if a named process has terminated.
void initd_process_wait(void);

// A list of all waiting sockets inside of this initd.
extern struct waiting_socket *waiting_socket_head;

// This adds a file descriptor to the list of waiting sockets. In the process
// this will free r.
struct waiting_socket *initd_waiting_socket_add(struct request *r);

// This disconnects an existing waiting socket. This will free w.
void initd_waiting_socket_disconnect(struct waiting_socket *w);

// ---------
// Main Loop
// ---------

// This is the main loop that will run select() and call out to all the other
// functions.
void initd_main_loop(int socket_fd, int signal_fd);

// --------
// Helpers.
// --------

// Close a file descriptor while accounting for possible EINTR returns.  This is
// a simple wrapper around close so it doesn't log or take any action beyond
// retrying if the syscall is interrupted.
int initd_close(int fd);

// Sets the given file descriptor as non blocking.
int initd_setnonblocking(int fd);

// Opens the given files as stdout and stderr, opens /dev/null as stdin, and
// closes all other open file descriptors. This is used inside of the forked
// process to safely execute customer code.
void initd_setup_fds(char *stdout_fn, char *stderr_fn);

// This will close all fds > 2, so it ignores stdin, stdout, and stderr.
void close_all_fds();

// Will print the time to current fd, e.g. stdout, stderr
void server_print_time(FILE *fd);

// Uses pivot_root to enter the root directory structure, chdir, and cleanup
// after itself. It uses pivot_root instead of chroot to ensure cleaner
// separation from the root mount namespace.
int pivot_root(char *root, bool privileged);

// -------
// Logging
// -------

// Set to 1 if debugging should be enabled.
extern bool cinitd_debugging;

#define DEBUG(...) \
	do { \
		if (cinitd_debugging) {			  \
			fflush(NULL);				  \
			server_print_time(stdout);			  \
			fprintf(stdout, __VA_ARGS__); \
			fflush(stdout);				  \
		} \
	} while(0)
#define INFO(...) \
	do {							  \
		server_print_time(stdout);			  \
		fprintf(stdout, __VA_ARGS__); \
		fflush(stdout);				  \
	} while(0)
#define ERROR(...) \
	do {							  \
		server_print_time(stderr);			  \
		fprintf(stderr, __VA_ARGS__); \
		fflush(stderr);				  \
	} while(0)
#define FATAL(...) \
	do {							  \
		server_print_time(stderr);			  \
		fprintf(stderr, __VA_ARGS__); \
		fflush(stderr);				  \
		exit(1)						  \
	} while(0)

// ------------
// Misc Helpers
// ------------

#define FREE(x) free(x)
#define CALLOC(x, y) calloc(x, y)

#endif
