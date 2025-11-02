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
	"runtime"
	"sync"
	"testing"

	apis "dirpx.dev/rfx/apis"
	"dirpx.dev/rfx/config"
	"dirpx.dev/rfx/registry"
)

// A few named types to avoid anonymous/unnamed pitfalls.
type T0 struct{}
type T1 struct{}
type T2 struct{}
type T3 struct{}
type T4 struct{}
type T5 struct{}
type T6 struct{}
type T7 struct{}
type T8 struct{}
type T9 struct{}

// TestConcurrentRegisterAndLookup verifies that Register/Lookup/Entries/Count
// are race-free and consistent under concurrent use.
func TestConcurrentRegisterAndLookup(t *testing.T) {
	cfg := config.DefaultConfig()
	reg := registry.New(cfg)

	types := []reflect.Type{
		reflect.TypeOf(T0{}), reflect.TypeOf(T1{}), reflect.TypeOf(T2{}),
		reflect.TypeOf(T3{}), reflect.TypeOf(T4{}), reflect.TypeOf(T5{}),
		reflect.TypeOf(T6{}), reflect.TypeOf(T7{}), reflect.TypeOf(T8{}),
		reflect.TypeOf(T9{}),
	}
	names := []string{"T0", "T1", "T2", "T3", "T4", "T5", "T6", "T7", "T8", "T9"}

	// Register once (sequential) to establish baseline.
	for i, tt := range types {
		if err := reg.Register(tt, names[i]); err != nil {
			t.Fatalf("register %s: %v", tt, err)
		}
	}

	// Hammer with concurrent lookups and idempotent re-registrations.
	wg := sync.WaitGroup{}
	workers := runtime.GOMAXPROCS(0) * 4

	// Readers
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < 5000; i++ {
				tt := types[i%len(types)]
				if got, ok := reg.Lookup(tt); !ok || got == "" {
					t.Errorf("lookup failed for %v: ok=%v got=%q", tt, ok, got)
					return
				}
				_ = reg.Count()
				_ = reg.Entries()
			}
		}()
	}

	// Writers (idempotent re-register)
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				j := (i + id) % len(types)
				_ = reg.Register(types[j], names[j]) // must be safe & idempotent
			}
		}(w)
	}

	wg.Wait()

	// Final consistency checks.
	if reg.Count() != len(types) {
		t.Fatalf("count mismatch: got %d want %d", reg.Count(), len(types))
	}
	got := map[reflect.Type]string{}
	for _, e := range reg.Entries() {
		got[e.Type] = e.Name
	}
	for i, tt := range types {
		if got[tt] != names[i] {
			t.Fatalf("entry mismatch for %v: got %q want %q", tt, got[tt], names[i])
		}
	}
}

// TestResetSnapshot ensures Reset is safe and Entries returns a stable snapshot.
func TestResetSnapshot(t *testing.T) {
	reg := registry.New(config.DefaultConfig())

	_ = reg.Register(reflect.TypeOf(T0{}), "T0")
	_ = reg.Register(reflect.TypeOf(T1{}), "T1")

	snap := reg.Entries() // snapshot copy expected
	reg.Reset()

	// After Reset, Count() should be 0, but previous snapshot must still be usable.
	if reg.Count() != 0 {
		t.Fatalf("count after reset: got %d want 0", reg.Count())
	}
	if len(snap) != 2 {
		t.Fatalf("snapshot length changed unexpectedly: %d", len(snap))
	}
	// sanity
	if snap[0].Name == "" || snap[1].Name == "" {
		t.Fatalf("snapshot contents invalid after reset")
	}
}

// This ensures the interface is satisfied; not a test but a compile-time check.
var _ apis.Registry = registry.New(config.DefaultConfig())
