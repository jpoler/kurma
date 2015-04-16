// Copyright 2015 Apcera Inc. All rights reserved.

package init

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/apcera/kurma/remote"
	"github.com/apcera/kurma/stage1/container"
	"github.com/apcera/kurma/stage1/server"
	"github.com/apcera/kurma/util"
	"github.com/apcera/logray"
	"github.com/apcera/util/proc"
	"github.com/vishvananda/netlink"
)

// loadConfigurationFile loads the configuration for the process. It will load
// the default configuration settings, the disk based configuration, and the
// command line based parameters. These get merged together to form the runtime
// values.
func (r *runner) loadConfigurationFile() error {
	f, err := os.Open(configurationFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to load configuration: %v", err)
	}
	defer f.Close()

	var diskConfig *kurmaConfig
	if err := json.NewDecoder(f).Decode(&diskConfig); err != nil {
		return fmt.Errorf("failed to parse configuration file: %v", err)
	}
	r.config.mergeConfig(diskConfig)

	cmdlineConfig := getConfigFromCmdline()
	r.config.mergeConfig(cmdlineConfig)
	return nil
}

// configureLogging is used to enable tracing logging, if it is turned on in the
// configuration.
func (r *runner) configureLogging() error {
	if r.config.Debug {
		logray.ResetDefaultLogLevel(logray.ALL)
	}
	return nil
}

// createSystemMounts configured the default mounts for the host. Since kurma is
// running as PID 1, there is no /etc/fstab, therefore it must mount them
// itself.
func (r *runner) createSystemMounts() error {
	// Default mounts to handle on boot. Note that order matters, they should be
	// alphabetical by mount location. Elements are: mount location, source,
	// fstype.
	systemMounts := [][]string{
		[]string{"/dev", "devtmpfs", "devtmpfs"},
		[]string{"/dev/pts", "none", "devpts"},
		[]string{"/proc", "none", "proc"},
		[]string{"/sys", "none", "sysfs"},
		[]string{"/tmp", "none", "tmpfs"},
		[]string{kurmaPath, "none", "tmpfs"},
		[]string{mountPath, "none", "tmpfs"},

		// put cgroups in a tmpfs so we can create the subdirectories
		[]string{cgroupsMount, "none", "tmpfs"},
	}

	r.log.Info("Creating system mounts")

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
			r.log.Tracef("- skipping %q, already mounted", location)
			continue
		}

		// perform the mount
		r.log.Tracef("- mounting %q (type %q) to %q", source, fstype, location)
		if err := handleMount(source, location, fstype, ""); err != nil {
			return fmt.Errorf("failed to mount %q: %v", location, err)
		}
	}
	return nil
}

// configureEnvironment sets environment variables that will be necessary for
// the process.
func (r *runner) configureEnvironment() error {
	os.Setenv("TMPDIR", "/tmp")
	os.Setenv("PATH", "/bin:/sbin")
	return nil
}

// mountCgroups handles creating the individual cgroup endpoints that are
// necessary.
func (r *runner) mountCgroups() error {
	// Default cgroups to mount and utilize.
	cgroupTypes := []string{
		"blkio",
		"cpu",
		"cpuacct",
		"devices",
		"memory",
	}

	r.log.Info("Setting up cgroups")

	// mount the cgroups
	for _, cgrouptype := range cgroupTypes {
		location := filepath.Join(cgroupsMount, cgrouptype)
		r.log.Tracef("- mounting cgroup %q to %q", cgrouptype, location)
		if err := handleMount("none", location, "cgroup", cgrouptype); err != nil {
			return fmt.Errorf("failed to mount cgroup %q: %v", cgrouptype, err)
		}

		// if this is the memory mount, need to set memory.use_hierarchy = 1
		if cgrouptype == "memory" {
			err := func() error {
				hpath := filepath.Join(location, "memory.use_hierarchy")
				f, err := os.OpenFile(hpath, os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
				if err != nil {
					return fmt.Errorf("Failed to configure memory hierarchy: %v", err)
				}
				defer f.Close()
				if _, err := f.WriteString("1\n"); err != nil {
					return fmt.Errorf("Failed to configure memory heirarchy: %v", err)
				}
				return nil
			}()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// loadModules handles loading all of the kernel modules that are specified in
// the configuration.
func (r *runner) loadModules() error {
	if len(r.config.Modules) == 0 {
		return nil
	}

	r.log.Infof("Loading specified modules [%s]", strings.Join(r.config.Modules, ", "))
	for _, mod := range r.config.Modules {
		if b, err := exec.Command("modprobe", mod).CombinedOutput(); err != nil {
			r.log.Errorf("- Failed to load module %q: %s", mod, string(b))
		}
	}
	return nil
}

// mountDisks handles walking the disk configuration to configure the specified
// disks, mount them, and make them accessible at the right locations.
func (r *runner) mountDisks() error {
	// Walk the disks to validate that usage entries aren't in multiple
	// records. Do this before making any changes to the disks.
	usages := make(map[kurmaPathUsage]bool, 0)
	for _, disk := range r.config.Disks {
		for _, u := range disk.Usage {
			if usages[u] {
				return fmt.Errorf("multiple disk entries cannot specify the same usage [%s]", string(u))
			}
			usages[u] = true
		}
	}

	// do the stuff
	for _, disk := range r.config.Disks {
		device := util.ResolveDevice(disk.Device)
		if device == "" {
			continue
		}
		fstype, _ := util.GetFsType(device)

		// FIXME check fstype against currently supported types

		// format it, if needed
		if fstype == "" || fstype != disk.FsType {
			r.log.Infof("Formatting disk %s to %s", device, disk.FsType)
			if err := formatDisk(device, disk.FsType); err != nil {
				r.log.Errorf("failed to format disk %q: %v", device, err)
				continue
			}
		}

		// mount it
		diskPath := filepath.Join(mountPath, strings.Replace(device, "/", "_", -1))
		if err := handleMount(device, diskPath, disk.FsType, ""); err != nil {
			r.log.Errorf("failed to mount disk: %v", err)
			continue
		}

		// setup usages
		for _, usage := range disk.Usage {
			usagePath := filepath.Join(diskPath, string(usage))

			// ensure the directory exists
			if err := os.MkdirAll(usagePath, os.FileMode(0755)); err != nil {
				r.log.Errorf("failed to create mount point: %v", err)
				continue
			}

			// bind mount it to the kurma path
			kurmaUsagePath := filepath.Join(kurmaPath, string(usage))
			if err := bindMount(usagePath, kurmaUsagePath); err != nil {
				r.log.Errorf("failed to bind mount the selected volume: %v", err)
				continue
			}
		}
	}
	return nil
}

// cleanOldPods removes the directories for any pods remaining from a previous
// run. If the host is booting up, those pods are obviously dead and stale.
func (r *runner) cleanOldPods() error {
	podsPath := filepath.Join(kurmaPath, string(kurmaPathPods))
	fis, err := ioutil.ReadDir(podsPath)
	if err != nil {
		r.log.Errorf("failed to check for existing pods: %v", err)
		return nil
	}

	for _, fi := range fis {
		if err := os.RemoveAll(filepath.Join(podsPath, fi.Name())); err != nil {
			r.log.Errorf("failed to cleanup existing pods: %v", err)
		}
	}
	return nil
}

// configureHostname calls to set the hostname to the one provided via
// configuration.
func (r *runner) configureHostname() error {
	if r.config.Hostname == "" {
		return nil
	}

	r.log.Infof("Setting hostname: %s", r.config.Hostname)
	if err := syscall.Sethostname([]byte(r.config.Hostname)); err != nil {
		r.log.Errorf("- Failed to set hostname: %v", err)
	}
	return nil
}

// configureNetwork handles iterating the local interfaces, matching it to an
// interface configuration, and configuring it. It will also handle configuring
// the default gateway after all interfaces are configured.
func (r *runner) configureNetwork() error {
	if r.config.NetworkConfig == nil {
		r.log.Warn("No network configuration given, skipping")
		return nil
	}

	r.log.Info("Configuring network...")

	links, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("failed to list network interfaces: %v", err)
	}

	for _, link := range links {
		linkName := link.Attrs().Name
		r.log.Debugf("Configuring %s...", linkName)

		// look for a matching network config entry
		var netconf *kurmaNetworkInterface
		for _, n := range r.config.NetworkConfig.Interfaces {
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
			r.log.Warn("- no matching network configuraton found")
			continue
		}

		// configure it
		if err := configureInterface(link, netconf); err != nil {
			r.log.Warnf("- %s", err.Error())
		}
	}

	// configure the gateway
	if r.config.NetworkConfig.Gateway != "" {
		gateway := net.ParseIP(r.config.NetworkConfig.Gateway)
		if gateway == nil {
			r.log.Warnf("Failed to configure gatway to %q", r.config.NetworkConfig.Gateway)
		}

		route := &netlink.Route{
			Scope: netlink.SCOPE_UNIVERSE,
			Gw:    gateway,
		}
		if err := netlink.RouteAdd(route); err != nil {
			r.log.Warnf("Failed to configure gateway: %v", err)
			return nil
		}
		r.log.Infof("Configured gatway to %s", r.config.NetworkConfig.Gateway)
	}

	// configure DNS
	if len(r.config.NetworkConfig.DNS) > 0 {
		// write the resolv.conf
		if err := os.RemoveAll("/etc/resolv.conf"); err != nil {
			r.log.Errorf("failed to cleanup old resolv.conf: %v", err)
			return nil
		}
		f, err := os.OpenFile("/etc/resolv.conf", os.O_CREATE, os.FileMode(0644))
		if err != nil {
			r.log.Errorf("failed to open /etc/resolv.conf: %v", err)
			return nil
		}
		defer f.Close()
		for _, ns := range r.config.NetworkConfig.DNS {
			if _, err := fmt.Fprintf(f, "nameserver %s\n", ns); err != nil {
				r.log.Errorf("failed to write to resolv.conf: %v", err)
				return nil
			}
		}
	}

	return nil
}

// createDirectories ensures the specified storage paths for pods and volumes
// exist.
func (r *runner) createDirectories() error {
	podsPath := filepath.Join(kurmaPath, string(kurmaPathPods))
	volumesPath := filepath.Join(kurmaPath, string(kurmaPathVolumes))

	if err := os.MkdirAll(podsPath, os.FileMode(0755)); err != nil {
		return fmt.Errorf("failed to create pods directory: %v", err)
	}
	if err := os.MkdirAll(volumesPath, os.FileMode(0755)); err != nil {
		return fmt.Errorf("failed to create volumes directory: %v", err)
	}
	return nil
}

// rootReadonly makes the root parition read only.
func (r *runner) rootReadonly() error {
	return syscall.Mount("", "/", "", syscall.MS_REMOUNT|syscall.MS_RDONLY, "")
}

func (r *runner) displayNetwork() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get all interfaces: %v", err)
	}

	r.log.Info(strings.Repeat("-", 30))
	defer r.log.Info(strings.Repeat("-", 30))
	r.log.Info("Network Information:")
	for _, in := range interfaces {
		ad, err := in.Addrs()
		if err != nil {
			return fmt.Errorf("failed to get addresses on interface %q: %v", in.Name, err)
		}
		addresses := make([]string, len(ad))
		for i, a := range ad {
			addresses[i] = a.String()
		}

		r.log.Infof("- %s: %s", in.Name, strings.Join(addresses, ", "))
	}
	return nil
}

// launchManager creates the container manager to allow containers to be
// launched.
func (r *runner) launchManager() error {
	mopts := &container.Options{
		ParentCgroupName:   r.config.ParentCgroupName,
		ContainerDirectory: filepath.Join(kurmaPath, string(kurmaPathPods)),
	}
	m, err := container.NewManager(mopts)
	if err != nil {
		return fmt.Errorf("failed to create the container manager: %v", err)
	}
	m.Log = r.log.Clone()
	r.manager = m
	r.log.Trace("Container Manager has been initialized.")

	os.Chdir("/var/kurma")
	return nil
}

// startSignalHandling configures the necessary signal handlers for the init
// process.
func (r *runner) startSignalHandling() error {
	// configure SIGCHLD
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGCHLD)
	go r.handleSIGCHLD(ch)
	return nil
}

// startServer begins the main Kurma RPC server and will take over execution.
func (r *runner) startServer() error {
	opts := &server.Options{
		ContainerManager: r.manager,
	}

	s := server.New(opts)
	go s.Start()
	return nil
}

// startInitContainers launches the initial containers that are specified in the
// configuration.
func (r *runner) startInitContainers() error {
	for _, img := range r.config.InitContainers {
		func() {
			f, err := remote.RetrieveImage(img)
			if err != nil {
				r.log.Errorf("Failed to retrieve image %q: %v", img, err)
				return
			}
			defer f.Close()

			manifest, err := findManifest(f)
			if err != nil {
				r.log.Errorf("Failed to find manifest in image %q: %v", img, err)
				return
			}

			if _, err := f.Seek(0, 0); err != nil {
				r.log.Errorf("Failed to set up %q: %v", img, err)
				return
			}

			r.manager.Create("", manifest, f)
			r.log.Infof("Launched container %s", manifest.Name.String())
		}()
	}
	return nil
}

// startUdev handles launching the udev service.
func (r *runner) startUdev() error {
	if !r.config.Services.Udev.Enabled {
		r.log.Trace("Skipping udev")
		return nil
	}

	f, err := remote.RetrieveImage(r.config.Services.Udev.ACI)
	if err != nil {
		r.log.Errorf("Failed to retrieve udev image: %v", err)
		return nil
	}
	defer f.Close()

	manifest, err := findManifest(f)
	if err != nil {
		r.log.Errorf("Failed to find manifest in udev image: %v", err)
		return nil
	}

	if _, err := f.Seek(0, 0); err != nil {
		r.log.Errorf("Failed to set up udev image: %v", err)
		return nil
	}

	container := r.manager.Create("udev", manifest, f)
	r.log.Debug("Started udev")

	container.Wait()
	r.log.Trace("Udev is finished")
	if err := container.Stop(); err != nil {
		r.log.Errorf("Failed to stop udev cleanly: %v", err)
		return nil
	}

	return nil
}

// startConsole handles launching the udev service.
func (r *runner) startNTP() error {
	if !r.config.Services.NTP.Enabled {
		r.log.Trace("Skipping NTP")
		return nil
	}

	r.log.Info("Updating system clock via NTP...")

	f, err := remote.RetrieveImage(r.config.Services.NTP.ACI)
	if err != nil {
		r.log.Errorf("Failed to retrieve console image: %v", err)
		return nil
	}
	defer f.Close()

	manifest, err := findManifest(f)
	if err != nil {
		r.log.Errorf("Failed to find manifest in console image: %v", err)
		return nil
	}

	if _, err := f.Seek(0, 0); err != nil {
		r.log.Errorf("Failed to set up console image: %v", err)
		return nil
	}

	// add the ntp servers on as environment variables
	manifest.App.Environment.Set(
		"NTP_SERVERS", strings.Join(r.config.Services.NTP.Servers, " "))

	r.manager.Create("ntp", manifest, f)
	r.log.Debug("Started NTP")
	return nil
}

// startConsole handles launching the udev service.
func (r *runner) startConsole() error {
	if !r.config.Services.Console.Enabled {
		r.log.Trace("Skipping console")
		return nil
	}

	f, err := remote.RetrieveImage(r.config.Services.Console.ACI)
	if err != nil {
		r.log.Errorf("Failed to retrieve console image: %v", err)
		return nil
	}
	defer f.Close()

	manifest, err := findManifest(f)
	if err != nil {
		r.log.Errorf("Failed to find manifest in console image: %v", err)
		return nil
	}

	if _, err := f.Seek(0, 0); err != nil {
		r.log.Errorf("Failed to set up console image: %v", err)
		return nil
	}

	// send in the configuration information
	manifest.App.Environment.Set(
		"CONSOLE_PASSWORD", r.config.Services.Console.Password)
	manifest.App.Environment.Set(
		"CONSOLE_KEYS", strings.Join(r.config.Services.Console.SSHKeys, "\n"))

	r.manager.Create("console", manifest, f)
	r.log.Debug("Started console")
	return nil
}
