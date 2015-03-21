// Copyright 2015 Apcera Inc. All rights reserved.

package bootstrap

type kurmaConfig struct {
	Datasources   []string            `json:"datasources,omitempty"`
	Hostname      string              `json:"hostname,omitempty"`
	NetworkConfig *kurmaNetworkConfig `json:"network_config,omitempty"`
	Modules       []string            `json:"modules,omitmepty"`
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
