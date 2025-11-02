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

package builder_test

import (
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	apis "dirpx.dev/rfx/apis"
	"dirpx.dev/rfx/builder"
	"dirpx.dev/rfx/config"
	"dirpx.dev/rfx/registry"
)

// userType is a plain named type with no special behavior.
// It is used to test fallback via reflection.
type userType struct{}

// hotType implements apis.Namer and is used to verify that the
// Namer-based strategy takes priority over other strategies.
type hotType struct{}

func (hotType) EntityName() string { return "hot-name" }

// defaultCfg returns a sane configuration for tests.
// It should match what a real process would use for normalization.
func defaultCfg() apis.Config {
	return apis.Config{
		IncludeBuiltins: true,
		MapPreferElem:   true,
		MaxUnwrap:       8,
	}
}

// TestBuildRegistry_Basic asserts that BuildRegistry returns a non-nil,
// working Registry that supports Register/Lookup/Entries/Count.
func TestBuildRegistry_Basic(t *testing.T) {
	b := builder.New()

	// prev may be nil; this must still produce a valid registry.
	reg := b.BuildRegistry(defaultCfg(), nil, nil)
	if reg == nil {
		t.Fatal("BuildRegistry returned nil")
	}

	tt := reflect.TypeOf(userType{})
	if err := reg.Register(tt, "userType"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if got, ok := reg.Lookup(tt); !ok || got != "userType" {
		t.Fatalf("Lookup mismatch: ok=%v got=%q want=%q", ok, got, "userType")
	}

	if c := reg.Count(); c < 1 {
		t.Fatalf("Count too small: %d", c)
	}

	snap := reg.Entries()
	if len(snap) < 1 {
		t.Fatalf("Entries returned empty snapshot")
	}
}

// TestBuildResolver_Order_NamerThenRegistryThenReflect verifies resolution priority:
// 1. If the value implements apis.Namer, use EntityName().
// 2. Otherwise, if the type is explicitly registered in the Registry, use that.
// 3. Otherwise, fall back to the reflect-based strategy ("pkg.Type").
func TestBuildResolver_Order_NamerThenRegistryThenReflect(t *testing.T) {
	b := builder.New()
	cfg := defaultCfg()

	// Build a fresh registry.
	reg := b.BuildRegistry(cfg, nil, nil)
	if reg == nil {
		t.Fatal("BuildRegistry returned nil")
	}

	// Register a type so the registry strategy can pick it up.
	type fromRegistry struct{}
	ttReg := reflect.TypeOf(fromRegistry{})
	if err := reg.Register(ttReg, "reg-name"); err != nil {
		t.Fatalf("Register(fromRegistry) failed: %v", err)
	}

	// Build resolver using that registry.
	res := b.BuildResolver(cfg, reg, nil, nil)
	if res == nil {
		t.Fatal("BuildResolver returned nil")
	}

	// (1) Namer should win.
	got := res.Resolve(hotType{}, cfg)
	if got != "hot-name" {
		t.Fatalf("Namer priority broken: got %q want %q", got, "hot-name")
	}

	// (2) Registry should be next.
	got = res.ResolveType(ttReg, cfg)
	if got != "reg-name" {
		t.Fatalf("Registry strategy broken: got %q want %q", got, "reg-name")
	}

	// (3) Reflect strategy is the fallback.
	ttUser := reflect.TypeOf(userType{})
	got = res.ResolveType(ttUser, cfg)
	if strings.TrimSpace(got) == "" {
		t.Fatalf("Reflect strategy returned empty name for userType")
	}
	if !strings.Contains(got, ".") {
		t.Fatalf("Reflect strategy name should contain a package prefix: %q", got)
	}
}

// TestBuildResolver_WithExternalRegistry asserts that BuildResolver will
// accept *any* apis.Registry implementation (not only the one created by
// this builder), and still resolve names from it.
func TestBuildResolver_WithExternalRegistry(t *testing.T) {
	// Create a registry directly using the package's public New().
	r := registry.New(config.DefaultConfig())

	if err := r.Register(reflect.TypeOf(userType{}), "u"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	res := builder.New().BuildResolver(defaultCfg(), r, nil, nil)
	if res == nil {
		t.Fatal("BuildResolver returned nil")
	}

	got := res.ResolveType(reflect.TypeOf(userType{}), defaultCfg())
	if got != "u" {
		t.Fatalf("resolver did not use registry mapping: got %q want %q", got, "u")
	}
}

// TestBuildResolver_Concurrency_Smoke hammers the resolver in parallel to ensure
// it is safe to call Resolve/ResolveType concurrently after being built.
func TestBuildResolver_Concurrency_Smoke(t *testing.T) {
	b := builder.New()
	cfg := defaultCfg()

	reg := b.BuildRegistry(cfg, nil, nil)
	if reg == nil {
		t.Fatal("BuildRegistry returned nil")
	}

	// Pre-register some names so the registry strategy and the namer strategy
	// both get exercised under contention.
	_ = reg.Register(reflect.TypeOf(userType{}), "userType")
	_ = reg.Register(reflect.TypeOf(hotType{}), "hotType") // Namer still should override

	res := b.BuildResolver(cfg, reg, nil, nil)
	if res == nil {
		t.Fatal("BuildResolver returned nil")
	}

	types := []reflect.Type{
		reflect.TypeOf(userType{}),
		reflect.TypeOf(hotType{}),
		reflect.TypeOf(&userType{}),
		reflect.TypeOf([]userType{}),
	}

	workers := runtime.GOMAXPROCS(0) * 4
	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				tt := types[(i+id)%len(types)]
				_ = res.ResolveType(tt, cfg)
				_ = res.Resolve(hotType{}, cfg)
			}
		}(w)
	}

	wg.Wait()
}

// Compile-time check: builder.New() must satisfy apis.Builder.
var _ apis.Builder = builder.New()
