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

package registry

import (
	"errors"
	"reflect"
	"sync"

	"dirpx.dev/rfx/apis"
	"dirpx.dev/rfx/config"
	uref "dirpx.dev/rfx/utils/reflect"
)

var (
	// ErrNilType is returned when a nil reflect.Type is provided.
	ErrNilType = errors.New("rfx(registry): nil reflect.Type provided")
	// ErrEmptyName is returned when an empty name is provided.
	ErrEmptyName = errors.New("rfx(registry): empty name provided")
	// ErrConflictingRegistration indicates an attempt to re-register
	// a type with a different name.
	ErrConflictingRegistration = errors.New("rfx(registry): conflicting type registration")
)

// New constructs a Registry that normalizes types according to cfg.
// Only MaxUnwrap and MapPreferElem are used here (IncludeBuiltins is irrelevant).
func New(cfg apis.Config) apis.Registry {
	if cfg.MaxUnwrap <= 0 {
		cfg.MaxUnwrap = config.DefaultMaxUnwrap
	}
	return &registry{cfg: cfg}
}

// registry is a simple Registry implementation backed by sync.Map.
type registry struct {
	// cfg is the configuration used for type normalization.
	cfg apis.Config
	// mu guards write-side consistency and counter
	mu sync.Mutex
	// m maps reflect.Type to registered name.
	m sync.Map // map[reflect.Type]string
	// count tracks the number of registered entries.
	count int
}

// Register associates the nearest named type of t with the given name.
// It is idempotent for the same (type,name) pair.
func (r *registry) Register(t reflect.Type, name string) error {
	// Validate inputs early.
	if t == nil {
		return ErrNilType
	}
	if name == "" {
		return ErrEmptyName
	}

	// Normalize to the nearest named type according to r.cfg.
	b, err := uref.Normalize(t, r.cfg)
	if err != nil {
		return err // ErrNotNamed (or ErrNilType if somehow nil sneaks in)
	}

	// Fast read path: idempotency / conflict check without locking.
	if old, ok := r.m.Load(b); ok {
		if old.(string) == name {
			return nil // idempotent re-registration
		}
		return ErrConflictingRegistration
	}

	// Write path: guard with a mutex to keep counter consistent and avoid ABA.
	r.mu.Lock()
	defer r.mu.Unlock()

	// Re-check under lock in case another goroutine stored meanwhile.
	if old, ok := r.m.Load(b); ok {
		if old.(string) == name {
			return nil
		}
		return ErrConflictingRegistration
	}

	r.m.Store(b, name)
	r.count++
	return nil
}

// Lookup returns a name for a type if present.
func (r *registry) Lookup(t reflect.Type) (name string, ok bool) {
	if t == nil {
		return "", false
	}
	nt, err := uref.Normalize(t, r.cfg)
	if err != nil {
		return "", false
	}
	if v, ok := r.m.Load(nt); ok {
		return v.(string), true
	}
	return "", false
}

// Entries returns a snapshot for diagnostics/docs (order is unspecified).
func (r *registry) Entries() []apis.Entry {
	entries := make([]apis.Entry, 0, r.Count())
	r.m.Range(func(key, value any) bool {
		entries = append(entries, apis.Entry{
			Type: key.(reflect.Type),
			Name: value.(string),
		})
		return true
	})
	return entries
}

// Count returns the number of registered entries.
func (r *registry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.count
}

// Reset clears all registered entries.
func (r *registry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m = sync.Map{}
	r.count = 0
}
