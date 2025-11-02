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

package strategy_test

import (
	"reflect"
	"runtime"
	"sync"
	"testing"

	apis "dirpx.dev/rfx/apis"
	rfxregistry "dirpx.dev/rfx/registry"
	"dirpx.dev/rfx/strategy"
)

// Local test types.
type A struct{}
type G[T any] struct{}

// cfg returns a convenient baseline Config for tests.
func cfg(opts ...func(*apis.Config)) apis.Config {
	c := apis.Config{
		IncludeBuiltins: true,
		MaxUnwrap:       8,
		MapPreferElem:   true,
	}
	for _, o := range opts {
		o(&c)
	}
	return c
}

func TestRegistryStrategy_WithRealRegistry_ByValue(t *testing.T) {
	conf := cfg()
	reg := rfxregistry.New(conf)

	// Register the named type "A" under a domain name.
	if err := reg.Register(reflect.TypeOf(A{}), "domain.A"); err != nil {
		t.Fatalf("Register(A): %v", err)
	}

	s := strategy.NewRegistryStrategy(reg)

	cases := []struct {
		name string
		val  any
		want string
	}{
		{"plain", A{}, "domain.A"},
		{"ptr", &A{}, "domain.A"},
		{"slice", []A{}, "domain.A"},
		{"array", [2]A{}, "domain.A"},
		{"chan", make(chan A), "domain.A"},
		{"map_prefer_elem", map[string]A{}, "domain.A"}, // Normalize(elem) -> A
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := s.TryResolve(tc.val, conf)
			if !ok || got != tc.want {
				t.Fatalf("TryResolve(%T) = (%q,%v), want (%q,true)", tc.val, got, ok, tc.want)
			}
		})
	}

	// Unknown type -> miss.
	if got, ok := s.TryResolve(G[int]{}, conf); ok || got != "" {
		t.Fatalf("TryResolve(G[int]{}) = (%q,%v), want ('',false)", got, ok)
	}
}

func TestRegistryStrategy_WithRealRegistry_ByType(t *testing.T) {
	conf := cfg()
	reg := rfxregistry.New(conf)

	if err := reg.Register(reflect.TypeOf(A{}), "domain.A"); err != nil {
		t.Fatalf("Register(A): %v", err)
	}

	s := strategy.NewRegistryStrategy(reg)

	cases := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{"type_plain", reflect.TypeOf(A{}), "domain.A"},
		{"type_ptr", reflect.TypeOf(&A{}), "domain.A"},
		{"type_slice", reflect.TypeOf([]A{}), "domain.A"},
		{"type_array", reflect.TypeOf([2]A{}), "domain.A"},
		{"type_chan", reflect.TypeOf((chan A)(nil)), "domain.A"},
		{"type_map_prefer_elem", reflect.TypeOf(map[string]A{}), "domain.A"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := s.TryResolveType(tc.typ, conf)
			if !ok || got != tc.want {
				t.Fatalf("TryResolveType(%v) = (%q,%v), want (%q,true)", tc.typ, got, ok, tc.want)
			}
		})
	}

	// Unknown type -> miss.
	if got, ok := s.TryResolveType(reflect.TypeOf(G[int]{}), conf); ok || got != "" {
		t.Fatalf("TryResolveType(G[int]) = (%q,%v), want ('',false)", got, ok)
	}
}

func TestRegistryStrategy_WithRealRegistry_MapPreferKey(t *testing.T) {
	// Prefer key: Normalize(map[K]V) returns K when both sides are named.
	conf := cfg(func(c *apis.Config) {
		c.MapPreferElem = false
		c.IncludeBuiltins = true // keep builtin "string" visible
	})
	reg := rfxregistry.New(conf)

	// Register "string" and "A" with different domain names to see which side wins.
	if err := reg.Register(reflect.TypeOf(""), "domain.string"); err != nil {
		t.Fatalf("Register(string): %v", err)
	}
	if err := reg.Register(reflect.TypeOf(A{}), "domain.A"); err != nil {
		t.Fatalf("Register(A): %v", err)
	}

	s := strategy.NewRegistryStrategy(reg)

	// With MapPreferElem=false, Normalize(map[string]A) -> string
	got, ok := s.TryResolveType(reflect.TypeOf(map[string]A{}), conf)
	if !ok || got != "domain.string" {
		t.Fatalf("prefer key: got (%q,%v), want (domain.string,true)", got, ok)
	}
}

// A small concurrency smoke test to ensure RegistryStrategy + real registry behave well.
func TestRegistryStrategy_WithRealRegistry_Concurrent(t *testing.T) {
	conf := cfg()
	reg := rfxregistry.New(conf)

	if err := reg.Register(reflect.TypeOf(A{}), "domain.A"); err != nil {
		t.Fatalf("Register(A): %v", err)
	}
	if err := reg.Register(reflect.TypeOf(""), "domain.string"); err != nil {
		t.Fatalf("Register(string): %v", err)
	}

	s := strategy.NewRegistryStrategy(reg)

	types := []reflect.Type{
		reflect.TypeOf(A{}),
		reflect.TypeOf(&A{}),
		reflect.TypeOf([]A{}),
		reflect.TypeOf(map[string]A{}),
		reflect.TypeOf(""),
	}
	want := []string{
		"domain.A",
		"domain.A",
		"domain.A",
		"domain.A",
		"domain.string",
	}

	workers := runtime.GOMAXPROCS(0) * 4
	iters := 2000

	var wg sync.WaitGroup
	wg.Add(workers)
	errCh := make(chan string, workers)

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				idx := i % len(types)
				got, ok := s.TryResolveType(types[idx], conf)
				if !ok || got != want[idx] {
					errCh <- got
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Fatalf("concurrent mismatch: got=%q", e)
	}
}
