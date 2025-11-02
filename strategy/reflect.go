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
	"path"
	"reflect"
	"strings"
	"sync"

	"dirpx.dev/rfx/apis"
	uref "dirpx.dev/rfx/utils/reflect"
)

// NewReflectStrategy creates an apis.Strategy that resolves names via reflection
// using utils/reflect.Normalize and memoization.
func NewReflectStrategy() apis.Strategy {
	return reflectStrategy{}
}

// reflectStrategy is the universal fallback that computes a stable "pkg.Type".
// It unwraps containers (ptr/slice/array/chan/map) via Normalize, strips generic
// instantiation parameters, and can hide builtin/no-package names.
type reflectStrategy struct{}

// Ensure reflectStrategy implements apis.Strategy.
var _ apis.Strategy = (*reflectStrategy)(nil)

// cacheKey ensures memoization respects all config knobs that affect resolution.
type cacheKey struct {
	t              reflect.Type
	includeBuiltin bool
	maxUnwrap      int16
	mapPreferElem  bool
}

// typeNameCache caches resolved type names by (type, config knobs).
var typeNameCache sync.Map // key: cacheKey, val: string

// TryResolve computes the domain-oriented name for v's type.
func (reflectStrategy) TryResolve(v any, cfg apis.Config) (string, bool) {
	if v == nil {
		return "", false
	}
	return byType(reflect.TypeOf(v), cfg), true
}

// TryResolveType computes the domain-oriented name for t.
func (reflectStrategy) TryResolveType(t reflect.Type, cfg apis.Config) (string, bool) {
	if t == nil {
		return "", false
	}
	return byType(t, cfg), true
}

// byType resolves the domain name for t with memoization.
func byType(t reflect.Type, cfg apis.Config) string {
	key := cacheKey{
		t:              t,
		includeBuiltin: cfg.IncludeBuiltins,
		maxUnwrap:      int16(cfg.MaxUnwrap),
		mapPreferElem:  cfg.MapPreferElem,
	}
	if v, ok := typeNameCache.Load(key); ok {
		return v.(string)
	}

	base, err := uref.Normalize(t, cfg)
	if err != nil || base == nil {
		typeNameCache.Store(key, "")
		return ""
	}

	name := stripTypeParams(base.Name())
	if p := base.PkgPath(); p != "" {
		name = path.Base(p) + "." + name
	} else if !cfg.IncludeBuiltins {
		// Hide builtin/no-package names if requested.
		name = ""
	}

	typeNameCache.Store(key, name)
	return name
}

// stripTypeParams removes generic type instantiation suffix: "T[int,string]" -> "T".
func stripTypeParams(s string) string {
	if i := strings.IndexByte(s, '['); i >= 0 {
		return s[:i]
	}
	return s
}
