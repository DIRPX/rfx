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

package rfx

import (
	"errors"
	"reflect"
	"sync"
	"sync/atomic"

	"dirpx.dev/rfx/apis"
	"dirpx.dev/rfx/builder"
	"dirpx.dev/rfx/config"
)

// init initializes the global res state.
func init() {
	// Initialize state with default cfg, reg, and res.
	s := &state{cfg: config.DefaultConfig()}
	b := builder.New()
	s.reg = b.BuildRegistry(s.cfg, nil, nil)
	s.res = b.BuildResolver(s.cfg, s.reg, nil, nil)
	s.bld = b
	// Store the initial state atomically.
	st.Store(s)
}

var (
	// ErrNilRegistry is returned when a builder returns a nil registry.
	ErrNilRegistry = errors.New("rfx: builder returned nil registry")
	// ErrNilResolver is returned when a builder returns a nil resolver.
	ErrNilResolver = errors.New("rfx: builder returned nil resolver")
)

// Entity resolves the name of the provided value v using the global rfx res.
// It uses the global rfx configuration and reg.
// This is a convenience wrapper around the global res.
func Entity(v any) string {
	s := st.Load()
	return s.res.Resolve(v, s.cfg)
}

// EntityType resolves the name of the provided reflect.Type t using the global rfx res.
// It uses the global rfx configuration and reg.
// This is a convenience wrapper around the global res.
func EntityType(t reflect.Type) string {
	s := st.Load()
	return s.res.ResolveType(t, s.cfg)
}

// RegisterType adds a type-name mapping to the global rfx reg.
// It uses the global rfx configuration.
// This is a convenience wrapper around the global reg.
func RegisterType(t reflect.Type, name string) error {
	return st.Load().reg.Register(t, name)
}

// SetAll explicitly sets all global rfx state components.
//
// Nil arguments leave the corresponding component unchanged,
// except for ext which is always replaced.
//
// This is a convenience wrapper around the global state.
func SetAll(cfg *apis.Config, ext any, reg apis.Registry, res apis.Resolver, bld apis.Builder) {
	buildMu.Lock()
	defer buildMu.Unlock()

	// Load the old state.
	old := st.Load()

	// Configuration
	ncfg := old.cfg
	if cfg != nil {
		ncfg = *cfg
	}

	// Extension
	next := ext

	// Builder
	nbld := old.bld
	if bld != nil {
		nbld = bld
	}

	// Registry
	nreg := reg
	npreg := false
	if nreg == nil {
		nreg = nbld.BuildRegistry(ncfg, old.reg, next)
	} else {
		npreg = true
	}

	// Resolver
	nres := res
	npres := false
	if nres == nil {
		nres = nbld.BuildResolver(ncfg, nreg, old.res, next)
	} else {
		npres = true
	}

	// Ensure non-nil reg and res.
	if nreg == nil {
		panic(ErrNilRegistry)
	}
	if nres == nil {
		panic(ErrNilResolver)
	}

	// Store the new state atomically.
	st.Store(
		&state{
			cfg:  ncfg,
			ext:  next,
			reg:  nreg,
			res:  nres,
			bld:  nbld,
			preg: npreg,
			pres: npres,
		},
	)
}

// Config returns the global rfx configuration.
func Config() apis.Config {
	return st.Load().cfg
}

// SetConfig sets the global rfx configuration to cfg.
// It rebuilds the global reg and res using the new configuration.
// This is a convenience wrapper around the global state.
func SetConfig(cfg apis.Config) {
	buildMu.Lock()
	defer buildMu.Unlock()

	// Load the old state.
	old := st.Load()
	b := old.bld

	// Build new nreg and res based on the new cfg and old state.
	nreg := old.reg
	if !old.preg {
		nreg = b.BuildRegistry(cfg, old.reg, old.ext)
	}
	nres := old.res
	if !old.pres {
		nres = b.BuildResolver(cfg, nreg, old.res, old.ext)
	}

	// Ensure non-nil nreg and res.
	if nreg == nil {
		panic(ErrNilRegistry)
	}
	if nres == nil {
		panic(ErrNilResolver)
	}

	// Store the new state atomically.
	st.Store(
		&state{
			cfg:  cfg,
			ext:  old.ext,
			reg:  nreg,
			res:  nres,
			bld:  b,
			preg: old.preg,
			pres: old.pres,
		},
	)
}

// Registry returns the global rfx reg.
func Registry() apis.Registry {
	return st.Load().reg
}

// SetRegistry sets the global rfx reg to reg.
// It uses the global rfx configuration to rebuild the global res.
// This is a convenience wrapper around the global state.
func SetRegistry(reg apis.Registry) {
	if reg == nil {
		return
	}

	buildMu.Lock()
	defer buildMu.Unlock()

	// Load the old state.
	old := st.Load()
	b := old.bld
	
	// Build new res based on the old cfg and new reg.
	nres := old.res
	if !old.pres {
		nres = b.BuildResolver(old.cfg, reg, old.res, old.ext)
	}

	// Ensure non-nil res.
	if nres == nil {
		panic(ErrNilResolver)
	}

	// Store the new state atomically.
	st.Store(
		&state{
			cfg:  old.cfg,
			ext:  old.ext,
			reg:  reg,
			res:  nres,
			bld:  b,
			preg: true,
			pres: old.pres,
		},
	)
}

// Resolver returns the global rfx res.
func Resolver() apis.Resolver {
	return st.Load().res
}

// SetResolver sets the global rfx res to res.
// It uses the global rfx configuration and reg.
// This is a convenience wrapper around the global state.
func SetResolver(res apis.Resolver) {
	if res == nil {
		return
	}

	buildMu.Lock()
	defer buildMu.Unlock()

	// Load the old state.
	old := st.Load()

	// Store the new state atomically.
	st.Store(
		&state{
			cfg:  old.cfg,
			ext:  old.ext,
			reg:  old.reg,
			res:  res,
			bld:  old.bld,
			preg: old.preg,
			pres: true,
		},
	)
}

// Builder returns the global rfx bld.
func Builder() apis.Builder {
	return st.Load().bld
}

// SetBuilder sets the global rfx bld to b.
// This is a convenience wrapper around the global state.
func SetBuilder(b apis.Builder) {
	if b == nil {
		return
	}

	buildMu.Lock()
	defer buildMu.Unlock()

	// Load the old state.
	old := st.Load()

	// Build new reg and res based on the new bld and old state.
	nreg := old.reg
	if !old.preg {
		nreg = b.BuildRegistry(old.cfg, old.reg, old.ext)
	}
	nres := old.res
	if !old.pres {
		nres = b.BuildResolver(old.cfg, nreg, old.res, old.ext)
	}

	// Ensure non-nil reg and res.
	if nreg == nil {
		panic(ErrNilRegistry)
	}
	if nres == nil {
		panic(ErrNilResolver)
	}

	// Store the new state atomically.
	st.Store(
		&state{
			cfg:  old.cfg,
			ext:  old.ext,
			reg:  nreg,
			res:  nres,
			bld:  b,
			preg: old.preg,
			pres: old.pres,
		},
	)
}

// SetExt replaces extension config and rebuilds non-pinned layers via the builder.
func SetExt[T any](ext T) {
	buildMu.Lock()
	defer buildMu.Unlock()

	// Load the old state.
	old := st.Load()
	b := old.bld

	// Build new reg and res based on the new ext and old state.
	nreg := old.reg
	if !old.preg {
		nreg = b.BuildRegistry(old.cfg, old.reg, ext)
	}
	nres := old.res
	if !old.pres {
		nres = b.BuildResolver(old.cfg, nreg, old.res, ext)
	}

	// Ensure non-nil reg and res.
	if nreg == nil {
		panic(ErrNilRegistry)
	}
	if nres == nil {
		panic(ErrNilResolver)
	}

	// Store the new state atomically.
	st.Store(
		&state{
			cfg:  old.cfg,
			ext:  ext,
			reg:  nreg,
			res:  nres,
			bld:  b,
			preg: old.preg,
			pres: old.pres,
		},
	)
}

// ExtAs returns the global rfx extension config as type T.
func ExtAs[T any]() (T, bool) {
	ext, ok := st.Load().ext.(T)
	return ext, ok
}

// IsRegistryPinned returns whether the global rfx reg is pinned (immutable).
func IsRegistryPinned() bool {
	return st.Load().preg
}

// PinRegistry makes the global rfx reg immutable.
func PinRegistry() {
	buildMu.Lock()
	defer buildMu.Unlock()

	// Load the old state.
	old := st.Load()

	// Store the new state atomically.
	st.Store(
		&state{
			cfg:  old.cfg,
			ext:  old.ext,
			reg:  old.reg,
			res:  old.res,
			bld:  old.bld,
			preg: true,
			pres: old.pres,
		},
	)
}

// UnpinRegistry makes the global rfx reg mutable again.
func UnpinRegistry() {
	buildMu.Lock()
	defer buildMu.Unlock()

	// Load the old state.
	old := st.Load()

	// Store the new state atomically.
	st.Store(
		&state{
			cfg:  old.cfg,
			ext:  old.ext,
			reg:  old.reg,
			res:  old.res,
			bld:  old.bld,
			preg: false,
			pres: old.pres,
		},
	)
}

// IsResolverPinned returns whether the global rfx res is pinned (immutable).
func IsResolverPinned() bool {
	return st.Load().pres
}

// PinResolver makes the global rfx res immutable.
func PinResolver() {
	buildMu.Lock()
	defer buildMu.Unlock()

	// Load the old state.
	old := st.Load()

	// Store the new state atomically.
	st.Store(
		&state{
			cfg:  old.cfg,
			ext:  old.ext,
			reg:  old.reg,
			res:  old.res,
			bld:  old.bld,
			preg: old.preg,
			pres: true,
		},
	)
}

// UnpinResolver makes the global rfx res mutable again.
func UnpinResolver() {
	buildMu.Lock()
	defer buildMu.Unlock()

	// Load the old state.
	old := st.Load()

	// Store the new state atomically.
	st.Store(
		&state{
			cfg:  old.cfg,
			ext:  old.ext,
			reg:  old.reg,
			res:  old.res,
			bld:  old.bld,
			preg: old.preg,
			pres: false,
		},
	)
}

// buildMu serializes writers (reconfigurations/swaps) so we never publish
// partially-built snapshots.
var buildMu sync.Mutex

// st is the global rfx state.
var st atomic.Pointer[state]

// state is the global rfx state snapshot.
// Immutable snapshot published atomically via st.Store; never mutate fields
// of a published state. Writers create a new state and swap it atomically.
type state struct {
	// cfg is the global rfx configuration.
	cfg apis.Config
	// ext is the global rfx extension configuration.
	ext any
	// reg is the global rfx reg.
	reg apis.Registry
	// res is the global rfx res.
	res apis.Resolver
	// bld is the global rfx bld.
	bld apis.Builder
	// preg indicates whether the reg is pinned (immutable).
	preg bool
	// pres indicates whether the res is pinned (immutable).
	pres bool
}
