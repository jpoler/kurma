// Copyright 2014-2015 Apcera Inc. All rights reserved.
//
// Portions of this file are based on code from:
//   https://github.com/arachsys/containers
//
// Code is licensed under MIT.
// Copyright 2013 Chris Webb <chris@arachsys.com>

#ifndef INITD_SPAWNER_CONSOLE_C
#define INITD_SPAWNER_CONSOLE_C

#define _GNU_SOURCE

#include <errno.h>
#include <fcntl.h>
#include <limits.h>
#include <poll.h>
#include <stdlib.h>
#include <termios.h>
#include <unistd.h>

#include <sys/ioctl.h>

#include "spawner.h"

static struct termios saved;

int getconsole(void) {
  int master;

  if ((master = posix_openpt(O_RDWR | O_NOCTTY)) < 0)
    error(1, 0, "Failed to allocate a console pseudo-terminal");
  grantpt(master);
  unlockpt(master);
  return master;
}

#endif
