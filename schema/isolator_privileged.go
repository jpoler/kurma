// Copyright 2015 Apcera Inc. All rights reserved.

package schema

import (
	"encoding/json"

	"github.com/appc/spec/schema/types"
)

const (
	PrivlegedName = "privileged"
)

func init() {
	types.AddIsolatorValueConstructor(PrivlegedName, newPrivileged)
}

func newPrivileged() types.IsolatorValue {
	n := Privileged(false)
	return &n
}

type Privileged bool

func (n *Privileged) UnmarshalJSON(b []byte) error {
	priv := false
	if err := json.Unmarshal(b, &priv); err != nil {
		return err
	}
	*n = Privileged(priv)
	return nil
}

func (n Privileged) AssertValid() error {
	return nil
}
