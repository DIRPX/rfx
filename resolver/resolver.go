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

package resolver

import (
	"reflect"

	"dirpx.dev/rfx/apis"
)

// New constructs an apis.Resolver that tries the given strategies in order.
// Nil strategies are ignored. The returned resolver is safe for concurrent use
// provided strategies themselves are safe for concurrent TryResolve calls.
func New(strategies ...apis.Strategy) apis.Resolver {
	// Filter out nils to avoid nil-interface panics on call sites.
	out := make([]apis.Strategy, 0, len(strategies))
	for _, s := range strategies {
		if s != nil {
			out = append(out, s)
		}
	}
	return chain{strats: out}
}

// chain is an immutable, order-preserving resolver over a set of strategies.
type chain struct {
	strats []apis.Strategy
}

// Resolve runs strategies in order until one handles the value.
// Returns an empty string if no strategy produced a name.
func (r chain) Resolve(v any, cfg apis.Config) string {
	for _, s := range r.strats {
		if name, ok := s.TryResolve(v, cfg); ok {
			return name
		}
	}
	return ""
}

// ResolveType runs strategies in order until one handles the type.
// Returns an empty string if no strategy produced a name.
func (r chain) ResolveType(t reflect.Type, cfg apis.Config) string {
	for _, s := range r.strats {
		if name, ok := s.TryResolveType(t, cfg); ok {
			return name
		}
	}
	return ""
}
