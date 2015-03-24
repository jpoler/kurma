// Copyright 2015 Apcera Inc. All rights reserved.

package schema

import (
	"encoding/json"

	"github.com/appc/spec/schema/types"
)

const (
	HostPrivlegedName = "host/privileged"
)

func init() {
	types.AddIsolatorValueConstructor(HostPrivlegedName, newHostPrivileged)
}

func newHostPrivileged() types.IsolatorValue {
	n := HostPrivileged(false)
	return &n
}

type HostPrivileged bool

func (n *HostPrivileged) UnmarshalJSON(b []byte) error {
	priv := false
	if err := json.Unmarshal(b, &priv); err != nil {
		return err
	}
	*n = HostPrivileged(priv)
	return nil
}

func (n HostPrivileged) AssertValid() error {
	return nil
}
