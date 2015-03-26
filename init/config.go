// Copyright 2015 Apcera Inc. All rights reserved.

package init

type kurmaConfig struct {
	Datasources      []string                  `json:"datasources,omitempty"`
	Hostname         string                    `json:"hostname,omitempty"`
	NetworkConfig    *kurmaNetworkConfig       `json:"network_config,omitempty"`
	Modules          []string                  `json:"modules,omitmepty"`
	Disks            []*kurmaDiskConfiguration `json:"disks,omitempty"`
	ParentCgroupName string                    `json:"parent_cgroup_name,omitempty"`
	InitContainers   []string                  `json:"init_containers,omitempty"`
}

type kurmaNetworkConfig struct {
	DNS        []string                 `json:"dns,omitempty"`
	Gateway    string                   `json:"gateway,omitempty"`
	Interfaces []*kurmaNetworkInterface `json:"interfaces,omitempty"`
}

type kurmaNetworkInterface struct {
	Device    string   `json:"device"`
	DHCP      bool     `json:"dhcp,omitmepty"`
	Address   string   `json:"address,omitempty"`
	Addresses []string `json:"addresses,omitempty"`
	MTU       int      `json:"mtu,omitmepty"`
}

type kurmaDiskConfiguration struct {
	Device string           `json:"device"`
	FsType string           `json:"fstype,omitempty"`
	Format *bool            `json:"format,omitempty"`
	Usage  []kurmaPathUsage `json:"usage"`
}

type kurmaPathUsage string

const (
	kurmaPathPods    = kurmaPathUsage("pods")
	kurmaPathVolumes = kurmaPathUsage("volumes")

	kurmaPath = "/var/kurma"
	mountPath = "/mnt"
)

func (cfg *kurmaConfig) mergeConfig(o *kurmaConfig) {
	// FIXME datasources

	// replace hostname
	if o.Hostname != "" {
		cfg.Hostname = o.Hostname
	}

	if o.NetworkConfig != nil {
		// replace dns
		if len(o.NetworkConfig.DNS) > 0 {
			cfg.NetworkConfig.DNS = o.NetworkConfig.DNS
		}
		// replace gateway
		if o.NetworkConfig.Gateway != "" {
			cfg.NetworkConfig.Gateway = o.NetworkConfig.Gateway
		}
		// replace interfaces
		if len(o.NetworkConfig.Interfaces) > 0 {
			cfg.NetworkConfig.Interfaces = o.NetworkConfig.Interfaces
		}
	}

	// append modules
	if len(o.Modules) > 0 {
		cfg.Modules = append(cfg.Modules, o.Modules...)
	}

	// replace disks
	if len(o.Disks) > 0 {
		cfg.Disks = o.Disks
	}

	// replace cgroup name
	if o.ParentCgroupName != "" {
		cfg.ParentCgroupName = o.ParentCgroupName
	}

	// append init containers
	if len(o.InitContainers) > 0 {
		cfg.InitContainers = append(cfg.InitContainers, o.InitContainers...)
	}
}
