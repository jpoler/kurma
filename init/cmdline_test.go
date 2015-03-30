// Copyright 2015 Apcera Inc. All rights reserved.
//
// This file is based on code from:
//   https://github.com/rancherio/os
//
// Code is licensed under Apache 2.0.
// Copyright (c) 2014-2015 Rancher Labs, Inc.

package init

import (
	"testing"

	tt "github.com/apcera/util/testtool"
)

func TestParseCmdline(t *testing.T) {
	expected := map[string]interface{}{
		"rescue":   true,
		"key1":     "value1",
		"key2":     "value2",
		"keyArray": []string{"1", "2"},
		"obj1": map[string]interface{}{
			"key3": "3value",
			"obj2": map[string]interface{}{
				"key4": true,
			},
		},
		"key5": 5,
	}

	actual := parseCmdline("a b kurma.rescue kurma.keyArray=[1,2] kurma.key1=value1 c kurma.key2=value2 kurma.obj1.key3=3value kurma.obj1.obj2.key4 kurma.key5=5")

	tt.TestEqual(t, actual, expected)
}
