// Copyright 2014-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SPAWNER_UTIL_C
#define INITD_SPAWNER_UTIL_C

#include <errno.h>
#include <fcntl.h>
#include <stdarg.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <sys/types.h>
#include <pwd.h>
#include <grp.h>

#include "spawner.h"

// Documented in cinitd.h
bool spawner_debugging = 0;

char *append(char **destination, const char *format, ...) {
	char *extra, *result;
	va_list args;

	va_start(args, format);
	if (vasprintf(&extra, format, args) < 0)
		error(1, errno, "asprintf");
	va_end(args);

	if (*destination == NULL) {
		*destination = extra;
		return extra;
	}

	if (asprintf(&result, "%s%s", *destination, extra) < 0)
			error(1, errno, "asprintf");
	free(*destination);
	free(extra);
	*destination = result;
	return result;
}

char *string(const char *format, ...) {
	char *result;
	va_list args;

	va_start(args, format);
	if (vasprintf(&result, format, args) < 0)
		error(1, errno, "asprintf");
	va_end(args);
	return result;
}

// Prints out the current timestamp.
void spawner_print_time(FILE* fd) {
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

void writemap(pid_t pid, char *type, char *map) {
	char *path;
	int fd;

	path = string("/proc/%d/%s_map", pid, type);
	if ((fd = open(path, O_WRONLY)) < 0)
		error(1, 0, "Failed to set container %s map", type);
	else if (write(fd, map, strlen(map)) != (ssize_t) strlen(map))
		error(1, 0, "Failed to set container %s map", type);
	free(path);
}

void waitforstop(pid_t child) {
	int status;

	if (waitpid(child, &status, WUNTRACED) < 0)
		error(1, errno, "waitpid");
	if (!WIFSTOPPED(status))
		exit(WEXITSTATUS(status));
}

void waitforexit(pid_t child) {
	int status;

	if (waitpid(child, &status, 0) < 0)
		error(1, errno, "waitpid");
	else if (WEXITSTATUS(status) != EXIT_SUCCESS)
		exit(WEXITSTATUS(status));
}

int uidforuser(char *user) {
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

int gidforgroup(char *group) {
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
