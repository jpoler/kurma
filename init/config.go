// Copyright 2015 Apcera Inc. All rights reserved.

package init

type kurmaConfig struct {
	Datasources      []string                `json:"datasources,omitempty"`
	Hostname         string                  `json:"hostname,omitempty"`
	NetworkConfig    *kurmaNetworkConfig     `json:"network_config,omitempty"`
	Modules          []string                `json:"modules,omitmepty"`
	Paths            *kurmaPathConfiguration `json:"paths,omitempty"`
	ParentCgroupName string                  `json:"parent_cgroup_name,omitempty"`
	InitContainers   []string                `json:"init_containers,omitempty"`
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

type kurmaPathConfiguration struct {
	Containers string `json:"containers,omitempty"`
}

func (cfg *kurmaConfig) mergeConfig(o *kurmaConfig) {
	// FIXME datasources

	if o.Hostname != "" {
		cfg.Hostname = o.Hostname
	}

	if o.NetworkConfig != nil {
		if len(o.NetworkConfig.DNS) > 0 {
			cfg.NetworkConfig.DNS = o.NetworkConfig.DNS
		}
		if o.NetworkConfig.Gateway != "" {
			cfg.NetworkConfig.Gateway = o.NetworkConfig.Gateway
		}
		if len(o.NetworkConfig.Interfaces) > 0 {
			cfg.NetworkConfig.Interfaces = o.NetworkConfig.Interfaces
		}
	}

	if len(o.Modules) > 0 {
		cfg.Modules = append(cfg.Modules, o.Modules...)
	}

	if o.Paths != nil {
		if o.Paths.Containers != "" {
			cfg.Paths.Containers = o.Paths.Containers
		}
	}

	if o.ParentCgroupName != "" {
		cfg.ParentCgroupName = o.ParentCgroupName
	}

	if len(o.InitContainers) > 0 {
		cfg.InitContainers = append(cfg.InitContainers, o.InitContainers...)
	}
}
