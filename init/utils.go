// Copyright 2015 Apcera Inc. All rights reserved.

package init

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/apcera/util/tarhelper"
	"github.com/appc/spec/schema"
	"github.com/vishvananda/netlink"
)

// handleMount takes care of creating the mount path and issuing the mount
// syscall for the mount source, location, and fstype.
func handleMount(source, location, fstype, data string) error {
	if err := os.MkdirAll(location, os.FileMode(0755)); err != nil {
		return err
	}
	return syscall.Mount(source, location, fstype, syscall.MS_MGC_VAL, data)
}

// configureInterface is used to configure an individual interface against a
// matched configuration. It sets up the addresses, the MTU, and invokes DHCP if
// necessary.
func configureInterface(link netlink.Link, netconf *kurmaNetworkInterface) error {
	linkName := link.Attrs().Name
	addressConfigured := true

	// FIXME DHCP
	// configure using DHCP
	if netconf.DHCP {
		cmd := exec.Command("/sbin/udhcpc", "-i", linkName, "-t", "20", "-n")
		cmd.Stdin = nil
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to configure %s with DHCP: %v", linkName, err)
		}
		addressConfigured = true
	}

	// single address
	if netconf.Address != "" {
		addr, err := netlink.ParseAddr(netconf.Address)
		if err != nil {
			return fmt.Errorf("failed to parse address %q on %s", netconf.Address, linkName)
		}
		if err := netlink.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("failed to configure address %q on %s: %v",
				netconf.Address, linkName, err)
		}
		addressConfigured = true
	}

	// list of addresses
	for _, address := range netconf.Addresses {
		addr, err := netlink.ParseAddr(address)
		if err != nil {
			return fmt.Errorf("failed to parse address %q on %s", address, linkName)
		}
		if err := netlink.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("failed to configure address %q on %s: %v",
				address, linkName, err)
		}
		addressConfigured = true
	}

	if !addressConfigured {
		return fmt.Errorf("no address configured to %s: unable to set link up", linkName)
	}

	if netconf.MTU > 0 {
		if err := netlink.LinkSetMTU(link, netconf.MTU); err != nil {
			return fmt.Errorf("failed to set mtu on %s: %v", linkName, err)
		}
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to set link %s up: %v", linkName, err)
	}

	return nil
}

// handleSIGCHLD is used to loop over and receive a SIGCHLD signal, which is
// used to have the process reap any dead child processes.
func (r *runner) handleSIGCHLD(ch chan os.Signal) {
	for _ = range ch {
		for {
			// This will loop until we're done reaping children. It'll call wait4, but
			// not block. If no processes are there to clean up, then it'll break the
			// loop and wait for a new signal.
			pid, err := syscall.Wait4(-1, nil, syscall.WNOHANG, nil)
			if err != nil {
				r.log.Errorf("Error in wait4: %v", err)
				break
			}
			if pid <= 0 {
				break
			}
		}
	}
}

// findManifest retrieves the manifest from the provided reader and unmarshals
// it.
func findManifest(r io.Reader) (*schema.ImageManifest, error) {
	arch, err := tarhelper.DetectArchiveCompression(r)
	if err != nil {
		return nil, err
	}

	for {
		header, err := arch.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("failed to locate manifest file")
		}
		if err != nil {
			return nil, err
		}

		if filepath.Clean(header.Name) != "manifest" {
			continue
		}

		var manifest *schema.ImageManifest
		if err := json.NewDecoder(arch).Decode(&manifest); err != nil {
			return nil, err
		}
		return manifest, nil
	}
}
