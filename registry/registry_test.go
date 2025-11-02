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

package registry_test

import (
	"reflect"
	"testing"

	"dirpx.dev/rfx/config"
	"dirpx.dev/rfx/registry"
)

func TestRegister_IdempotentAndLookup(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := registry.New(cfg)

	// pointer -> nearest named = T1
	err := reg.Register(reflect.TypeOf(&T1{}), "domain.T1")
	if err != nil {
		t.Fatalf("Register(&T1{}): unexpected error: %v", err)
	}
	// idempotent re-register with same name
	if err := reg.Register(reflect.TypeOf(&T1{}), "domain.T1"); err != nil {
		t.Fatalf("Register(&T1{}) idempotent: unexpected error: %v", err)
	}

	// lookup by exact type
	if name, ok := reg.Lookup(reflect.TypeOf(&T1{})); !ok || name != "domain.T1" {
		t.Fatalf("Lookup(&T1{}): got (%q,%v), want (domain.T1,true)", name, ok)
	}
	// lookup by elem/slice/etc should hit the same base
	if name, ok := reg.Lookup(reflect.TypeOf([]T1{})); !ok || name != "domain.T1" {
		t.Fatalf("Lookup([]T1{}): got (%q,%v), want (domain.T1,true)", name, ok)
	}

	if reg.Count() != 1 {
		t.Fatalf("Count() = %d, want 1", reg.Count())
	}
}

func TestRegister_Conflict(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := registry.New(cfg)

	if err := reg.Register(reflect.TypeOf(&T1{}), "domain.T1"); err != nil {
		t.Fatalf("Register: unexpected error: %v", err)
	}
	// Same normalized type (nearest named T1), different name -> conflict
	err := reg.Register(reflect.TypeOf([]*T1{}), "other.Name")
	if err == nil || err != registry.ErrConflictingRegistration {
		t.Fatalf("expected ErrConflictingRegistration, got: %v", err)
	}
}

func TestRegister_Errors(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := registry.New(cfg)

	if err := reg.Register(nil, "x"); err != registry.ErrNilType {
		t.Fatalf("nil type: want ErrNilType, got %v", err)
	}
	if err := reg.Register(reflect.TypeOf(&T1{}), ""); err != registry.ErrEmptyName {
		t.Fatalf("empty name: want ErrEmptyName, got %v", err)
	}
}

func TestNormalize_MaxUnwrapLimit(t *testing.T) {
	// Set MaxUnwrap = 1 so **T1 fails to reach named type
	cfg := config.DefaultConfig()
	cfg.MaxUnwrap = 1
	reg := registry.New(cfg)

	// **T1 -> after 1 unwrap stays *T1 (Ptr, unnamed), should error NotNamed on Register
	type PtrPtrT1 = **T1
	var x PtrPtrT1
	_ = reg.Register(reflect.TypeOf(x), "domain.T1")

	// With enough unwraps it should succeed
	cfg2 := config.DefaultConfig()
	cfg2.MaxUnwrap = 8
	reg2 := registry.New(cfg2)
	if err := reg2.Register(reflect.TypeOf(x), "domain.T1"); err != nil {
		t.Fatalf("MaxUnwrap=8: unexpected error: %v", err)
	}
}

func TestMapPreference_ElementVsKey(t *testing.T) {
	// Prefer element (default): map[string]T2 -> nearest named = T2
	cfgElem := config.DefaultConfig()
	cfgElem.MapPreferElem = true
	regElem := registry.New(cfgElem)

	mapType := reflect.TypeOf(map[string]T2{})
	if err := regElem.Register(mapType, "domain.T2"); err != nil {
		t.Fatalf("Register(map[string]T2): %v", err)
	}
	if name, ok := regElem.Lookup(mapType); !ok || name != "domain.T2" {
		t.Fatalf("Lookup(map[string]T2): got (%q,%v), want (domain.T2,true)", name, ok)
	}

	// Prefer key: map[string]T2 -> nearest named = string (builtin is "named" by reflect)
	cfgKey := config.DefaultConfig()
	cfgKey.MapPreferElem = false
	regKey := registry.New(cfgKey)

	if err := regKey.Register(mapType, "builtin.string"); err != nil {
		t.Fatalf("Register(map[string]T2) prefer key: %v", err)
	}
	if name, ok := regKey.Lookup(mapType); !ok || name != "builtin.string" {
		t.Fatalf("Lookup(map[string]T2) prefer key: got (%q,%v), want (builtin.string,true)", name, ok)
	}
}

func TestEntriesAndReset(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := registry.New(cfg)

	_ = reg.Register(reflect.TypeOf(&T1{}), "domain.T1")
	_ = reg.Register(reflect.TypeOf(&T2{}), "domain.T2")

	entries := reg.Entries()
	if len(entries) != 2 {
		t.Fatalf("Entries len = %d, want 2", len(entries))
	}
	if reg.Count() != 2 {
		t.Fatalf("Count() = %d, want 2", reg.Count())
	}

	reg.Reset()

	if reg.Count() != 0 {
		t.Fatalf("after Reset, Count() = %d, want 0", reg.Count())
	}
	if name, ok := reg.Lookup(reflect.TypeOf(&T1{})); ok || name != "" {
		t.Fatalf("Lookup after Reset: got (%q,%v), want ('',false)", name, ok)
	}
}

func TestLookupNilAndUnknown(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := registry.New(cfg)

	if name, ok := reg.Lookup(nil); ok || name != "" {
		t.Fatalf("Lookup(nil): got (%q,%v), want ('',false)", name, ok)
	}
	if name, ok := reg.Lookup(reflect.TypeOf(&T1{})); ok || name != "" {
		t.Fatalf("Lookup(unknown): got (%q,%v), want ('',false)", name, ok)
	}
}
