// Copyright 2015 Apcera Inc. All rights reserved.
//
// Portions of this file are based on code from:
//   https://github.com/rancherio/os
//
// Code is licensed under Apache 2.0.
// Copyright (c) 2014-2015 Rancher Labs, Inc.

package init

import (
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
)

func getConfigFromCmdline() *kurmaConfig {
	b, err := ioutil.ReadFile("/proc/self/cmdline")
	if err != nil {
		return nil
	}
	parsed := parseCmdline(string(b))

	config, _ := uglyDoubleLoop(parsed)
	return config
}

func uglyDoubleLoop(genericConfig map[string]interface{}) (*kurmaConfig, error) {
	b, err := json.Marshal(genericConfig)
	if err != nil {
		return nil, err
	}
	var config *kurmaConfig
	if err := json.Unmarshal(b, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func parseCmdline(cmdLine string) map[string]interface{} {
	result := make(map[string]interface{})

outer:
	for _, part := range strings.Split(cmdLine, " ") {
		if !strings.HasPrefix(part, "kurma.") {
			continue
		}

		var value string
		kv := strings.SplitN(part, "=", 2)

		if len(kv) == 1 {
			value = "true"
		} else {
			value = kv[1]
		}

		current := result
		keys := strings.Split(kv[0], ".")[1:]
		for i, key := range keys {
			if i == len(keys)-1 {
				current[key] = dummyMarshall(value)
			} else {
				if obj, ok := current[key]; ok {
					if newCurrent, ok := obj.(map[string]interface{}); ok {
						current = newCurrent
					} else {
						continue outer
					}
				} else {
					newCurrent := make(map[string]interface{})
					current[key] = newCurrent
					current = newCurrent
				}
			}
		}
	}
	return result
}

func dummyMarshall(value string) interface{} {
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		return strings.Split(value[1:len(value)-1], ",")
	}

	if value == "true" {
		return true
	} else if value == "false" {
		return false
	} else if ok, _ := regexp.MatchString("^[0-9]+$", value); ok {
		i, err := strconv.Atoi(value)
		if err != nil {
			panic(err)
		}
		return i
	}

	return value
}
