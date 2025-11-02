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

package builder

import (
	"dirpx.dev/rfx/apis"
	"dirpx.dev/rfx/registry"
	"dirpx.dev/rfx/resolver"
	"dirpx.dev/rfx/strategy"
)

// New creates and returns a new instance of an apis.Builder.
func New() apis.Builder {
	return &builder{}
}

// builder is an empty struct to be used as a receiver for builder methods.
type builder struct{}

// BuildRegistry builds and returns a new apis.Registry based on the provided configuration
// and pre-existing registry. If a pre-existing registry is provided, its entries are copied
// into the new registry.
func (b *builder) BuildRegistry(cfg apis.Config, preg apis.Registry, _ any) apis.Registry {
	nreg := registry.New(cfg)
	if preg != nil {
		for _, e := range preg.Entries() {
			_ = nreg.Register(e.Type, e.Name)
		}
	}
	return nreg
}

// BuildResolver builds and returns a new apis.Resolver based on the provided configuration,
// registry, and pre-existing resolver. If a pre-existing resolver is provided, its state
// may be reused in the new resolver.
func (b *builder) BuildResolver(cfg apis.Config, reg apis.Registry, _ apis.Resolver, _ any) apis.Resolver {
	return resolver.New(
		strategy.NewNamerStrategy(),
		strategy.NewRegistryStrategy(reg),
		strategy.NewReflectStrategy(),
	)
}
