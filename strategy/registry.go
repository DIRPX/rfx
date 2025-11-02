/*
   Copyright 2025 The DIRPX Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package strategy

import (
	"reflect"

	"dirpx.dev/rfx/apis"
)

// NewRegistryStrategy creates a strategy.Strategy that uses a rfx.Registry.
func NewRegistryStrategy(reg apis.Registry) apis.Strategy {
	return &registryStrategy{reg: reg}
}

// registryStrategy consults a provided rfx.Registry (reflection-free lookup).
type registryStrategy struct {
	reg apis.Registry
}

// Ensure registryStrategy implements strategy.Strategy.
var _ apis.Strategy = (*registryStrategy)(nil)

// TryResolve looks up v's type in the registry.
func (s *registryStrategy) TryResolve(v any, _ apis.Config) (string, bool) {
	if v == nil || s.reg == nil {
		return "", false
	}
	return s.reg.Lookup(reflect.TypeOf(v))
}

// TryResolveType looks up t in the registry.
func (s *registryStrategy) TryResolveType(t reflect.Type, _ apis.Config) (string, bool) {
	if t == nil || s.reg == nil {
		return "", false
	}
	return s.reg.Lookup(t)
}
