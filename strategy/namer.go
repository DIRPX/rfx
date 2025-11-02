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

// NewNamerStrategy creates an apis.Strategy that uses rfx.Namer.
func NewNamerStrategy() apis.Strategy {
	return &namerStrategy{}
}

// namerStrategy is a zero-cost fast path: if v implements rfx.Namer,
// return its EntityName() and stop the chain.
type namerStrategy struct{}

// Ensure NamerStrategy implements apis.Strategy.
var _ apis.Strategy = (*namerStrategy)(nil)

// TryResolve checks if v implements rfx.Namer and returns its EntityName().
func (*namerStrategy) TryResolve(v any, _ apis.Config) (string, bool) {
	if v == nil {
		return "", false
	}
	if n, ok := v.(apis.Namer); ok {
		return n.EntityName(), true
	}
	return "", false
}

// TryResolveType always returns false: Namer requires an instance.
func (*namerStrategy) TryResolveType(_ reflect.Type, _ apis.Config) (string, bool) {
	// No instance -> cannot use Namer.
	return "", false
}
