// Copyright 2013 Apcera Inc. All rights reserved.

package envmap

import (
	"fmt"
	"os"
)

// Provides a simple storage layer for environment like variables.
type EnvMap struct {
	env     map[string]string
	parent  *EnvMap
	flatten bool
}

func NewEnvMap() (r *EnvMap) {
	r = new(EnvMap)
	r.env = make(map[string]string, 0)
	r.flatten = true
	return r
}

// FlattenMap when set to false will not flatten the
// results of an EnvMap.
func (e *EnvMap) FlattenMap(flatMap bool) {
	e.flatten = flatMap
}

func (e *EnvMap) Set(key, value string) {
	if prev, ok := e.env[key]; ok == true {
		resolve := func(s string) string {
			if s == key {
				return prev
			}
			return "$" + key
		}
		e.env[key] = os.Expand(value, resolve)
	} else {
		e.env[key] = value
	}
}

func (e *EnvMap) get(
	key string, top *EnvMap, processQueue map[string]*EnvMap,
	cache map[string]string,
) (string, bool) {
	resolve := func(s string) string {
		if value, ok := cache[s]; ok == true {
			return value
		}
		if last, ok := processQueue[s]; ok == true {
			// If this is the last element in this environment map
			// then we return ""
			if last == nil {
				return ""
			}
			processQueue[s] = last.parent
			r, _ := last.get(s, top, processQueue, cache)
			return r
		}
		processQueue[s] = top
		r, _ := top.get(s, top, processQueue, cache)
		return r
	}

	for e != nil {
		if value, ok := e.env[key]; ok == true {
			processQueue[key] = e.parent
			s := os.Expand(value, resolve)
			delete(processQueue, key)
			return s, true
		}
		e = e.parent
	}
	return "", false
}

func (e *EnvMap) Get(key string) (string, bool) {
	// This is used to ensure that we do not recurse forever while
	// attempting to get a variable.
	processQueue := make(map[string]*EnvMap, 10)
	cache := make(map[string]string, 1)
	return e.get(key, e, processQueue, cache)
}

func (e *EnvMap) GetRaw(key string) (string, bool) {
	for e != nil {
		if value, ok := e.env[key]; ok == true {
			return value, true
		}
		e = e.parent
	}
	return "", false
}

func (e *EnvMap) Map() map[string]string {
	cache := make(map[string]string, len(e.env))
	processQueue := make(map[string]*EnvMap, 10)

	for p := e; p != nil; p = p.parent {
		for k := range p.env {
			if _, ok := cache[k]; ok == false {
				if e.flatten {
					cache[k], _ = e.get(k, e, processQueue, cache)
				} else {
					cache[k], _ = e.GetRaw(k)
				}
			}
		}
	}

	return cache
}

func (e *EnvMap) Strings() []string {
	m := e.Map()
	r := make([]string, 0, len(m))
	for k, v := range m {
		r = append(r, fmt.Sprintf("%s=%s", k, v))
	}
	return r
}

func (e *EnvMap) NewChild() *EnvMap {
	return &EnvMap{
		env:     make(map[string]string, 0),
		parent:  e,
		flatten: true,
	}
}
