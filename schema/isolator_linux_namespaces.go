// Copyright 2015 Apcera Inc. All rights reserved.

package schema

import (
	"encoding/json"
	"fmt"

	"github.com/appc/spec/schema/types"
)

const (
	LinuxNamespacesName = "os/linux/namespaces"

	nsIPC   = "ipc"
	nsMount = "mount"
	nsNet   = "net"
	nsPID   = "pid"
	nsUser  = "user"
	nsUTS   = "uts"
)

func init() {
	types.AddIsolatorValueConstructor(LinuxNamespacesName, newLinuxNamespace)
}

func newLinuxNamespace() types.IsolatorValue {
	return &LinuxNamespaces{
		ns: make(map[string]bool),
	}
}

type LinuxNamespaces struct {
	ns map[string]bool
}

func (n *LinuxNamespaces) UnmarshalJSON(b []byte) error {
	var namespaces []string
	if err := json.Unmarshal(b, &namespaces); err != nil {
		return err
	}

	for _, namespace := range namespaces {
		n.ns[namespace] = true
	}
	return nil
}

func (n *LinuxNamespaces) AssertValid() error {
	for k, _ := range n.ns {
		switch k {
		case nsIPC, nsMount, nsNet, nsPID, nsUser, nsUTS:
		default:
			return fmt.Errorf("unrecognized namespace %q", k)
		}
	}
	return nil
}

func (n *LinuxNamespaces) IPC() bool {
	return n.ns[nsIPC]
}

func (n *LinuxNamespaces) Mount() bool {
	return n.ns[nsMount]
}

func (n *LinuxNamespaces) Net() bool {
	return n.ns[nsNet]
}

func (n *LinuxNamespaces) PID() bool {
	return n.ns[nsPID]
}

func (n *LinuxNamespaces) User() bool {
	return n.ns[nsUser]
}

func (n *LinuxNamespaces) UTS() bool {
	return n.ns[nsUTS]
}
