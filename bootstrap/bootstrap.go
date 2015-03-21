// Copyright 2015 Apcera Inc. All rights reserved.

package bootstrap

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/apcera/logray"
	"github.com/apcera/util/proc"
	"github.com/vishvananda/netlink"
)

var (
	// The setup functions that should be run in order to handle setting up the
	// host system to create and manage containers. These functions focus
	// primarily on runtime actions that must be done each time on boot.
	setupFunctions = []func() error{
		createSystemMounts,
		mountCgroups,
		loadConfiguration,
		loadModules,
		configureHostname,
		configureNetwork,
		displayNetwork,
	}

	// The logger is set in Run() so that it inherits any default outputs. If it
	// is set in init(), it won't have them.
	log *logray.Logger
)

const (
	// The default location where cgroups should be mounted. This is a constant
	// because it is referenced in multiple functions.
	cgroupsMount = "/sys/fs/cgroup"
)

var config *kurmaConfig

// Run handles executing the bootstrap setup. This prepares the current host
// environment to run and manage containers. It will return an error if any part
// of the setup fails.
func Run() error {
	log = logray.New()
	log.Info("Running bootstrap")

	for _, f := range setupFunctions {
		if err := f(); err != nil {
			log.Errorf("ERROR: %v", err)
			return err
		}
	}
	return nil
}

// createSystemMounts configured the default mounts for the host. Since kurma is
// running as PID 1, there is no /etc/fstab, therefore it must mount them
// itself.
func createSystemMounts() error {
	// Default mounts to handle on boot. Note that order matters, they should be
	// alphabetical by mount location. Elements are: mount location, source,
	// fstype.
	systemMounts := [][]string{
		[]string{"/dev", "devtmpfs", "devtmpfs"},
		[]string{"/dev/pts", "none", "devpts"},
		[]string{"/proc", "none", "proc"},
		[]string{"/sys", "none", "sysfs"},

		// put cgroups in a tmpfs so we can create the subdirectories
		[]string{cgroupsMount, "none", "tmpfs"},
	}

	log.Info("Creating system mounts")

	// Check if the /proc/mounts file exists to see if there are mounts that
	// already exist. This is primarily to support testing bootstrapping with
	// kurma launched by kurma (yes, meta)
	var existingMounts map[string]*proc.MountPoint
	if _, err := os.Lstat(proc.MountProcFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to check if %q existed: %v", proc.MountProcFile, err)
	} else if os.IsNotExist(err) {
		// really are freshly booted, /proc isn't mounted, so make this blank
		existingMounts = make(map[string]*proc.MountPoint)
	} else {
		// Get existing mount points.
		existingMounts, err = proc.MountPoints()
		if err != nil {
			return fmt.Errorf("failed to read existing mount points: %v", err)
		}
	}

	for _, mount := range systemMounts {
		location, source, fstype := mount[0], mount[1], mount[2]

		// check if it exists
		if _, exists := existingMounts[location]; exists {
			log.Tracef("- skipping %q, already mounted", location)
			continue
		}

		// perform the mount
		log.Tracef("- mounting %q (type %q) to %q", source, fstype, location)
		if err := handleMount(source, location, fstype, ""); err != nil {
			return fmt.Errorf("failed to mount %q: %v", location, err)
		}
	}
	return nil
}

// mountCgroups handles creating the individual cgroup endpoints that are
// necessary.
func mountCgroups() error {
	// Default cgroups to mount and utilize.
	cgroupTypes := []string{
		"blkio",
		"cpu",
		"cpuacct",
		"devices",
		"memory",
	}

	log.Info("Setting up cgroups")

	for _, cgrouptype := range cgroupTypes {
		location := filepath.Join(cgroupsMount, cgrouptype)
		log.Tracef("- mounting cgroup %q to %q", cgrouptype, location)
		if err := handleMount("none", location, "cgroup", cgrouptype); err != nil {
			return fmt.Errorf("failed to mount cgroup %q: %v", cgrouptype, err)
		}
	}
	return nil
}

// loadConfiguration handles loading the initial configuration from the what is
// available at initial boot time.
func loadConfiguration() error {
	config = &kurmaConfig{
		Hostname: "kurmaos",
		Modules:  []string{"e1000"},
		NetworkConfig: &kurmaNetworkConfig{
			Interfaces: []*kurmaNetworkInterface{
				&kurmaNetworkInterface{
					Device:  "lo",
					Address: "127.0.0.1/8",
				},
				&kurmaNetworkInterface{
					Device: "eth.+",
					DHCP:   true,
				},
			},
		},
	}
	return nil
}

// loadModules handles loading all of the kernel modules that are specified in
// the configuration.
func loadModules() error {
	if len(config.Modules) == 0 {
		return nil
	}

	log.Infof("Loading specified modules [%s]", strings.Join(config.Modules, ", "))
	for _, mod := range config.Modules {
		if err := exec.Command("/sbin/modprobe", mod).Run(); err != nil {
			log.Errorf("- Failed to load module: %s", mod)
		}
	}
	return nil
}

// configureHostname calls to set the hostname to the one provided via
// configuration.
func configureHostname() error {
	if config.Hostname == "" {
		return nil
	}

	log.Infof("Setting hostname: %s", config.Hostname)
	if err := syscall.Sethostname([]byte(config.Hostname)); err != nil {
		log.Errorf("- Failed to set hostname: %v", err)
	}
	return nil
}

// configureNetwork handles iterating the local interfaces, matching it to an
// interface configuration, and configuring it. It will also handle configuring
// the default gateway after all interfaces are configured.
func configureNetwork() error {
	if config.NetworkConfig == nil {
		log.Warn("No network configuration given, skipping")
		return nil
	}

	links, err := netlink.LinkList()
	if err != nil {
		return err
	}

	for _, link := range links {
		linkName := link.Attrs().Name
		log.Infof("Configuring %s...", linkName)

		// look for a matching network config entry
		var netconf *kurmaNetworkInterface
		for _, n := range config.NetworkConfig.Interfaces {
			if linkName == n.Device {
				netconf = n
				break
			}
			if match, _ := regexp.MatchString(n.Device, linkName); match {
				netconf = n
				break
			}
		}

		// handle if none are found
		if netconf == nil {
			log.Warn("- no matching network configuraton found")
			continue
		}

		// configure it
		if err := configureInterface(link, netconf); err != nil {
			log.Warnf("- %s", err.Error())
		}
	}

	// configure the gateway
	if config.NetworkConfig.Gateway != "" {
		gateway := net.ParseIP(config.NetworkConfig.Gateway)
		if gateway == nil {
			log.Warnf("Failed to configure gatway to %q", config.NetworkConfig.Gateway)
		}

		route := &netlink.Route{
			Scope: netlink.SCOPE_UNIVERSE,
			Gw:    gateway,
		}
		if err := netlink.RouteAdd(route); err != nil {
			log.Warnf("Failed to configure gateway: %v", err)
			return nil
		}
		log.Infof("Configured gatway to %s", config.NetworkConfig.Gateway)
	}

	return nil
}

// configureInterface is used to configure an individual interface against a
// matched configuration. It sets up the addresses, the MTU, and invokes DHCP if
// necessary.
func configureInterface(link netlink.Link, netconf *kurmaNetworkInterface) error {
	linkName := link.Attrs().Name

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

func displayNetwork() error {
	fmt.Printf("INTERFACES:\n")

	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	for _, in := range interfaces {
		fmt.Printf("\t%#v\n", in)
		ad, err := in.Addrs()
		if err != nil {
			return err
		}
		for _, a := range ad {
			fmt.Printf("\t\taddr: %v\n", a)
		}
	}
	return nil
}
