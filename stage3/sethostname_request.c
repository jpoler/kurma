// Copyright 2013-2015 Apcera Inc. All rights reserved.

#ifndef INITD_SERVER_SETHOSTNAME_REQUEST_C
#define INITD_SERVER_SETHOSTNAME_REQUEST_C

#include <errno.h>
#include <string.h>
#include <unistd.h>

#include "cinitd.h"

// Documented in cinitd.h
void sethostname_request(struct request *r) {
  // The expected protocol for a sethostname statement looks like this:
  // {
  //   { "SETHOSTNAME" "HOSTNAME" },
  // }

  INFO("[%d] SETHOSTNAME request.\n", r->fd);

  // Protocol error conditions.
  if (
    (r->outer_len != 1) ||
    // SETHOSTNAME
    (r->data[0][1] == NULL) ||
    (r->data[0][2] != NULL) ||
    // END
    (r->data[1] != NULL))
  {
    INFO("[%d] Protocol error.\n", r->fd);
    initd_response_protocol_error(r);
    return;
  }

  // Attempt to set the hostname now.
  if (sethostname(r->data[0][1], strlen(r->data[0][1])) != 0) {
    ERROR("[%d] Failed to sethostname('%s', %zu): %s\n", r->fd, r->data[0][1], strlen(r->data[0][1]), strerror(errno));
    initd_response_internal_error(r);
    return;
  }

  // Success. Inform the caller.
  INFO("[%d] Successful sethostname('%s', %zu), responding OK.\n", r->fd, r->data[0][1], strlen(r->data[0][1]));
  initd_response_request_ok(r);
}

#endif
