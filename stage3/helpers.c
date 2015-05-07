// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_HELPERS_C
#define INITD_SERVER_HELPERS_C

#include <dirent.h>
#include <errno.h>
#include <fcntl.h>
#include <grp.h>
#include <limits.h>
#include <paths.h>
#include <pwd.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <sysexits.h>

#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/syscall.h>
#include <sys/types.h>

#include "cinitd.h"

// Documented in cinitd.h
bool cinitd_debugging = 0;

// Documented in cinitd.h
int initd_close(int fd)
{
	while (close(fd)) {
		if (errno != EINTR) {
			return -1;
		}
	}
	return 0;
}

// Documented in cinitd.h
int initd_setnonblocking(int fd)
{
	int flags;

	flags = fcntl(fd, F_GETFL, 0);
	if (flags == -1) {
		ERROR("[%d] Error in fcntl(F_GETFL): %s\n", fd, strerror(errno));
		return 1;
	}

	flags |= O_NONBLOCK;
	if (fcntl(fd, F_SETFL, flags)) {
		ERROR("[%d] Error in fcntl(F_SETFL): %s\n", fd, strerror(errno));
		return 1;
	}

	return 0;
}

int STDERR = 2;

void close_all_fds()
{
	char buffer[sizeof(struct dirent) + PATH_MAX + 1];
	DIR *d;
	int closed;
	int i;
	struct dirent *results;

	DEBUG("close_all_fds\n");

	do {
		closed = 0;

		// Open the directory. This shouldn't ever fail since the directory should
		// always exist.
		d = opendir("/proc/self/fdinfo");
		if (d == NULL) {
			ERROR("Could not opendir\n");
			_exit(EX_OSERR);
		}

		while (true) {
			// Read an element from the directory. This should represent a file
			// descriptor in its name. Errors here are fatal.
			if (readdir_r(d, (struct dirent *) &buffer, &results) != 0) {
				ERROR("Exiting, could not readdir\n");
				_exit(EX_OSERR);
			}

			// NULL here represents the end of the directory stream.
			if (results == NULL) { break; }

			// Parse the file name into a number.
			i = atoi(results->d_name);

			// If the number is 0, 1, or 2 then we can't close it (thats our stdin,
			// stdout, and stderr), which we will replace below with the proper dup2
			// call, and if it has the file descriptor of the directory we are
			// iterating then we can't close that either.  In both cases we skip this
			// directory entry.
			if (i <= STDERR || i == dirfd(d)) {
				DEBUG("Skipping %d\n", i);
				continue;
			}

			// Close the file descriptor that we just read. Note we need to retry in
			// the case where the close() operation is interrupted.
			DEBUG("Closing FD[%d]\n", i);

			if (initd_close(i)) {
				ERROR("Exiting, could not close FD[%d] - %s\n", i, strerror(errno));
				_exit(EX_OSERR);
			}
			closed++;
		}

		// Close the directory file descriptor.
		if (closedir(d) == -1) {
			ERROR("Exiting, could not closedir(d)\n");
			_exit(EX_OSERR);
		}

		// Repeat the loop until we can iterate over top of it without closing
		// anything.
	} while (closed != 0);
}

// Documented in cinitd.h
void initd_setup_fds(char *stdout_fn, char *stderr_fn)
{
	int fd;
	int flags;
	int flags_dev_null;
	int mode;

	DEBUG("initd_setup_fds\n");

	// Flags used for opening stdout and stderr. Special for /dev/null.  Don't
	// append to logs; just overwrite any files that we already see. This can
	// happen when we're creating a capsule or an app from a snapshot that
	// persisted files from a previous exec. That's why O_APPEND & O_EXCL are not
	// here. (anymore)
	flags = O_WRONLY | O_CREAT | O_NOFOLLOW;

	// Explicitly force the mode to be 0700, ignoring umask. We want these logs to
	// be accessible only to root/root since it is not clear what will be written
	// to them.
	mode = 0700;

	// stdin
	fd = open(_PATH_DEVNULL, O_RDONLY | O_NOFOLLOW, mode);
	if (fd == -1) {
		ERROR("Exiting, could not open new stdin\n");
		_exit(EX_OSERR);
	}
	if (dup2(fd, 0) == -1) {
		ERROR("Exiting, error duping stdin fd, %d\n", fd);
		_exit(EX_OSERR);
	}
	if (initd_close(fd)) {
		ERROR("Exiting, error closing stdin fd, %d\n", fd);
		_exit(EX_OSERR);
	}

	// stdout
	//	initd_close(1);
	if (stdout_fn == NULL || !strcmp(stdout_fn, _PATH_DEVNULL)) {
		fd = open(_PATH_DEVNULL, O_WRONLY | O_APPEND | O_NOFOLLOW, mode);
	} else {
		fd = open(stdout_fn, flags, mode);
	}
	if (fd == -1) {
		ERROR("Exiting, could not open new stdout: %s - %s\n", stdout_fn, strerror(errno));
		_exit(EX_OSERR);
	}
	if (dup2(fd, 1) == -1) {
		ERROR("Exiting, error duping stdout fd, %d\n", fd);
		_exit(EX_OSERR);
	}
	if (initd_close(fd)) {
		ERROR("Exiting, error closing stdout fd, %d\n", fd);
		_exit(EX_OSERR);
	}

	// stderr
	if (stderr_fn == NULL || !strcmp(stderr_fn, _PATH_DEVNULL)) {
		fd = open(_PATH_DEVNULL, O_WRONLY | O_APPEND | O_NOFOLLOW, mode);
	} else {
		fd = open(stderr_fn, flags, mode);
	}
	if (fd == -1) {
		ERROR("Exiting, could not open new stderr: %s - %s\n", stdout_fn, strerror(errno));
		_exit(EX_OSERR);
	}
	if (dup2(fd, 2) == -1) {
		ERROR("Exiting, error duping stderr fd, %d\n", fd);
		_exit(EX_OSERR);
	}
	if (initd_close(fd)) {
		ERROR("Exiting, error closing stdout fd, %d\n", fd);
		_exit(EX_OSERR);
	}
}

// Documented in cinitd.h
int pivot_root(char *root, bool privileged) {
	if (chdir(root) < 0)
		return -1;
	if (mkdir("host", 0755) < 0)
		return -1;
	if (syscall(__NR_pivot_root, ".", "host") < 0)
		return -1;
	if (chdir("/") < 0)
		return -1;

	if (!privileged) {
		if(umount2("/host", MNT_DETACH) < 0)
			return -1;
		rmdir("/host");
	}
	return 0;
}

int uidforuser2(char *user) {
	// First, look up the /etc/passwd entry.
	struct passwd *pwd;
	pwd = getpwnam(user);
	if (pwd != NULL)
		return pwd->pw_uid;

	// Second, attempt to convert to integer first
	char *endptr;
	long val;
	errno = 0;
	val = strtol(user, &endptr, 10);

	// if the whole thing matched, return it
	if (*endptr == '\0') {
		return (int) val;
	}

	return -1;
}

int gidforgroup2(char *group) {
	// First, look up the /etc/group entry
	struct group *grp;
	grp = getgrnam(group);
	if (grp != NULL)
		return grp->gr_gid;

	// Second, attempt to convert to integer first
	char *endptr;
	long val;
	errno = 0;
	val = strtol(group, &endptr, 10);

	// if the whole thing matched, return it
	if (*endptr == '\0') {
		return (int) val;
	}

	return -1;
}

#endif
