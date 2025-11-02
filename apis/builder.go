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

package apis

// Builder composes Registry and Resolver from a Config.
// Implementations may migrate state from previous instances (prev*), or ignore them.
type Builder interface {
	// BuildRegistry constructs a Registry for Config. May migrate entries from previous registry.
	// ext is an optional extension context. Its meaning is implementation-defined.
	BuildRegistry(cfg Config, reg Registry, ext any) Registry
	// BuildResolver constructs a Resolver for Config and Registry. May reuse state from previous resolver.
	// ext is an optional extension context. Its meaning is implementation-defined.
	BuildResolver(cfg Config, reg Registry, res Resolver, ext any) Resolver
}
