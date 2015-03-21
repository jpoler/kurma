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
