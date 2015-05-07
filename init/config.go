// Copyright 2015 Apcera Inc. All rights reserved.

package init

type kurmaConfig struct {
	Debug              bool                      `json:"debug,omitempty"`
	OEMConfig          *OEMConfig                `json:"oem_config"`
	Datasources        []string                  `json:"datasources,omitempty"`
	Hostname           string                    `json:"hostname,omitempty"`
	NetworkConfig      kurmaNetworkConfig        `json:"network_config,omitempty"`
	Modules            []string                  `json:"modules,omitmepty"`
	Disks              []*kurmaDiskConfiguration `json:"disks,omitempty"`
	ParentCgroupName   string                    `json:"parent_cgroup_name,omitempty"`
	RequiredNamespaces []string                  `json:"required_namespaces,omitempty"`
	Services           kurmaServices             `json:"services,omitempty"`
	InitContainers     []string                  `json:"init_containers,omitempty"`
}

type OEMConfig struct {
	Device     string `json:"device"`
	ConfigPath string `json:"config_path"`
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
	Resize bool             `json:"resize"`
}

type kurmaPathUsage string

const (
	kurmaPathPods    = kurmaPathUsage("pods")
	kurmaPathVolumes = kurmaPathUsage("volumes")

	kurmaPath = "/var/kurma"
	mountPath = "/mnt"
)

type kurmaServices struct {
	NTP     kurmaNTPService     `json:"ntp,omitempty"`
	Udev    kurmaGenericService `json:"udev,omitempty"`
	Console kurmaConsoleService `json:"console,omitempty"`
}

type kurmaGenericService struct {
	Enabled *bool  `json:"enabled,omitempty"`
	ACI     string `json:"aci,omitempty"`
}

type kurmaNTPService struct {
	Enabled  *bool    `json:"enabled,omitempty"`
	ACI      string   `json:"aci,omitempty"`
	Servers  []string `json:"servers,omitempty"`
	Interval string   `json:"interval,omitempty"`
}

type kurmaConsoleService struct {
	Enabled  *bool    `json:"enabled,omitempty"`
	ACI      string   `json:"aci,omitempty"`
	Password *string  `json:"password,omitmepty"`
	SSHKeys  []string `json:"ssh_keys,omitempty"`
}

func (cfg *kurmaConfig) mergeConfig(o *kurmaConfig) {
	if o == nil {
		return
	}

	// FIXME datasources

	// oem config
	if o.OEMConfig != nil {
		cfg.OEMConfig = o.OEMConfig
	}

	// replace hostname
	if o.Hostname != "" {
		cfg.Hostname = o.Hostname
	}

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

	// NTP
	if o.Services.NTP.Enabled != nil {
		cfg.Services.NTP.Enabled = o.Services.NTP.Enabled
	}
	if o.Services.NTP.ACI != "" {
		cfg.Services.NTP.ACI = o.Services.NTP.ACI
	}
	if len(o.Services.NTP.Servers) > 0 {
		cfg.Services.NTP.Servers = o.Services.NTP.Servers
	}

	// Udev
	if o.Services.Udev.Enabled != nil {
		cfg.Services.Udev.Enabled = o.Services.Udev.Enabled
	}
	if o.Services.Udev.ACI != "" {
		cfg.Services.Udev.ACI = o.Services.Udev.ACI
	}

	// Console
	if o.Services.Console.Enabled != nil {
		cfg.Services.Console.Enabled = o.Services.Console.Enabled
	}
	if o.Services.Console.ACI != "" {
		cfg.Services.Console.ACI = o.Services.Console.ACI
	}
	if o.Services.Console.Password != nil {
		cfg.Services.Console.Password = o.Services.Console.Password
	}
	if len(o.Services.Console.SSHKeys) > 0 {
		cfg.Services.Console.SSHKeys = o.Services.Console.SSHKeys
	}
}
