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

import (
	"reflect"
)

// Strategy is a pluggable resolution step. A Resolver can chain multiple
// strategies in order (e.g., Namer -> Registry -> Reflect).
type Strategy interface {
	// TryResolve attempts to resolve a name for value v according to cfg.
	// It returns (name, true) if handled; otherwise ("" , false) to fall through.
	TryResolve(v any, cfg Config) (name string, handled bool)

	// TryResolveType attempts to resolve a name for the reflect.Type t.
	TryResolveType(t reflect.Type, cfg Config) (name string, handled bool)
}
