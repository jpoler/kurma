// Copyright 2014-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SPAWNER_STARTUP_C
#define INITD_SPAWNER_STARTUP_C

#include <getopt.h>
#include <stdlib.h>

#include "spawner.h"

// This is a basic C wrapper that is used to intercept the running process if
// its PID 1, which can only happen if this is the initd for a new
// container. This code _MUST_ execute before the golang source starts.  In
// order to do this we use a construct known to work in GCC and clang which
// allows a specific set of code to execute before main(). This is a C
// "constructor".
//
// With this constructor we can ensure that we run our cspawner() function prior
// to golang startup for cgroups_initd.
void cspawner(int argc, char **argv) __attribute__ ((constructor));

void usage(char *progname) {
	fprintf(stderr, "\
Usage: %s [OPTIONS] DIR [CMD [ARG]...]\n\
", progname);
	exit(64);
}

static int verbose_flag;

// This function is executed prior to golang's startup logic.
void cspawner(int argc, char **argv) {
	pid_t child, parent;
	clone_destination_data *args;
	static int new_ipc_namespace, new_mount_namespace, new_network_namespace,
	new_pid_namespace, new_uts_namespace, new_user_namespace, detach, chroot;
	int c;
	char **tmp;

	// ensure we're intended to run
	if (getenv("SPAWNER_INTERCEPT") == NULL) {
		return;
	}

	// Enable debugging if necessary.
	if (getenv("SPAWNER_DEBUG") != NULL) {
		spawner_debugging = true;
		DEBUG("Spawner debugging logs enabled.\n");
	}

	// initialize our args
	args = (clone_destination_data *) calloc(1, sizeof(clone_destination_data));

	// environment to pass to the child
	args->environment = NULL;
	size_t env_len = 0;

	// tasksfiles, used for cgroups
	args->tasksfiles = NULL;
	size_t tasksfiles_len = 0;

	// initialize the fd args to -1 so we know when they weren't specified
	args->stdinfd = -1;
	args->stdoutfd = -1;
	args->stderrfd = -1;

	// loop and process the arguments
	while(1) {
		static struct option long_options[] =
			{
				{"env", required_argument, 0, 'a'},
				{"taskfile", required_argument, 0, 'b'},

				{"stdinfd",  required_argument, 0, 'c'},
				{"stdoutfd", required_argument, 0, 'd'},
				{"stderrfd", required_argument, 0, 'e'},

				{"new-ipc-namespace", no_argument, &new_ipc_namespace, 1},
				{"new-mount-namespace", no_argument, &new_mount_namespace, 1},
				{"new-network-namespace", no_argument, &new_network_namespace, 1},
				{"new-pid-namespace", no_argument, &new_pid_namespace, 1},
				{"new-uts-namespace", no_argument, &new_uts_namespace, 1},
				{"new-user-namespace", no_argument, &new_user_namespace, 1},

				{"uidmap", required_argument, 0, 'k'},
				{"gidmap", required_argument, 0, 'l'},

				{"directory", required_argument, 0, 'm'},
				{"user", required_argument, 0, 'n'},
				{"group", required_argument, 0, 'o'},

				{"detach", no_argument, &detach, 1},
				{"chroot", no_argument, &chroot, 1},
				{0, 0, 0, 0}
			};
		/* getopt_long stores the option index here. */
		int option_index = 0;

		c = getopt_long(argc, argv, "abcdeklm", long_options, &option_index);

		/* Detect the end of the options. */
		if (c == -1)
			break;

		switch (c) {
		case 0:
			/* If this option set a flag, do nothing else now. */
			if (long_options[option_index].flag != 0)
			break;
			printf ("option %s", long_options[option_index].name);
			if (optarg)
			printf (" with arg %s", optarg);
			printf ("\n");
			break;

			// env
		case 'a':
			args->environment = realloc(args->environment, sizeof(char*) * (env_len+1));
			if (!args->environment) { error(1, 0, "environment was null"); }
			args->environment[env_len] = optarg;
			env_len++;
			break;

			// taskfile
		case 'b':
			args->tasksfiles = realloc(args->tasksfiles, sizeof(char*) * (tasksfiles_len+1));
			if (!args->tasksfiles) { error(1, 0, "tasksfiles was null"); }
			args->tasksfiles[tasksfiles_len] = optarg;
			tasksfiles_len++;
			break;

			// fds
		case 'c':
			args->stdinfd = atoi(optarg);
			break;
		case 'd':
			args->stdoutfd = atoi(optarg);
			break;
		case 'e':
			args->stderrfd = atoi(optarg);
			break;

			// maps
		case 'k':
			args->uidmap = optarg;
			break;
		case 'l':
			args->gidmap = optarg;
			break;

			// directory
		case 'm':
			args->container_directory = optarg;
			break;
			// user
		case 'n':
			args->user = optarg;
			break;
			// group
		case 'o':
			args->group = optarg;
			break;

		case '?':
			/* getopt_long already printed an error message. */
			break;

		default:
			abort();
		}
	}

	// copy over the flags
	args->new_ipc_namespace = new_ipc_namespace;
	args->new_mount_namespace = new_mount_namespace;
	args->new_network_namespace = new_network_namespace;
	args->new_pid_namespace = new_pid_namespace;
	args->new_uts_namespace = new_uts_namespace;
	args->new_user_namespace = new_user_namespace;
	args->detach = detach;
	args->chroot = chroot;

	// ensure the final element in arrays is null
	args->environment = realloc(args->environment, sizeof(char*) * (env_len+1));
	args->environment[env_len] = NULL;
	args->tasksfiles = realloc(args->tasksfiles, sizeof(char*) * (tasksfiles_len+1));
	args->tasksfiles[tasksfiles_len] = NULL;

	// populate the command args
	args->command = argv[optind];
	args->args = argv + optind;

	// launch the container
	DEBUG("Beginning spawning\n");
	spawn_child(args);

	// Ensure that we never ever fall back into the go world.
	exit(0);
}

#endif
