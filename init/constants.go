// Copyright 2015 Apcera Inc. All rights reserved.

package init

var (
	// The setup functions that should be run in order to handle setting up the
	// host system to create and manage containers. These functions focus
	// primarily on runtime actions that must be done each time on boot.
	setupFunctions = []func(*runner) error{
		(*runner).createSystemMounts,
		(*runner).loadConfigurationFile,
		(*runner).configureLogging,
		(*runner).configureEnvironment,
		(*runner).mountCgroups,
		(*runner).loadModules,
		(*runner).startSignalHandling,
		(*runner).launchManager,
		(*runner).createDirectories,
		(*runner).startUdev,
		(*runner).mountDisks,
		(*runner).cleanOldPods,
		(*runner).configureHostname,
		(*runner).configureNetwork,
		(*runner).rootReadonly,
		(*runner).startNTP,
		(*runner).startServer,
		(*runner).startInitContainers,
		(*runner).displayNetwork,
		(*runner).startConsole,
	}
)

const (
	// configurationFile is the source of the initial disk based configuration.
	configurationFile = "/etc/kurma.json"

	// The default location where cgroups should be mounted. This is a constant
	// because it is referenced in multiple functions.
	cgroupsMount = "/sys/fs/cgroup"
)

// defaultConfiguration returns the default codified configuration that is
// applied on boot.
func defaultConfiguration() *kurmaConfig {
	return &kurmaConfig{
		Hostname:           "kurmaos",
		ParentCgroupName:   "kurma",
		RequiredNamespaces: []string{"ipc", "mount", "pid", "uts"},
		NetworkConfig: &kurmaNetworkConfig{
			Interfaces: []*kurmaNetworkInterface{
				&kurmaNetworkInterface{
					Device:  "lo",
					Address: "127.0.0.1/8",
				},
			},
		},
		Services: &kurmaServices{
			NTP: &kurmaNTPService{
				Enabled: true,
				ACI:     "file:///ntp.aci",
				Servers: []string{
					"0.pool.ntp.org",
					"1.pool.ntp.org",
					"2.pool.ntp.org",
					"3.pool.ntp.org",
				},
			},
			Udev: &kurmaGenericService{
				Enabled: false,
				ACI:     "file:///udev.aci",
			},
			Console: &kurmaConsoleService{
				Enabled:  true,
				ACI:      "file:///console.aci",
				Password: "kurma",
			},
		},
	}
}
