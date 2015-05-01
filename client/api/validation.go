// Copyright 2015 Apcera Inc. All rights reserved.

package api

import (
	"fmt"

	kschema "github.com/apcera/kurma/schema"
	"github.com/appc/spec/schema"
)

func validateImageManifest(imageManifest *schema.ImageManifest) error {
	if imageManifest.App == nil {
		return fmt.Errorf("the imageManifest must specify an App")
	}

	// Reject any containers that request host privilege. This can only be started
	// with the local API, not remote API.
	if iso := imageManifest.App.Isolators.GetByName(kschema.HostPrivlegedName); iso != nil {
		if piso, ok := iso.Value().(*kschema.HostPrivileged); ok {
			if *piso {
				return fmt.Errorf("host privileged containers cannot be launched remotely")
			}
		}
	}

	// FIXME once network isolation is in, this should force adding the container
	// namespaces isolator to ensure any remotely sourced images are network
	// namespaced.

	return nil
}
