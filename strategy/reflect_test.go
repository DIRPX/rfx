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
	"runtime"
	"sync"
	"testing"

	"dirpx.dev/rfx/apis"
)

// Local test types.
type A struct{}
type G[T any] struct{}
type W[T any] struct{ V T }

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

func TestReflectStrategy_ByValue(t *testing.T) {
	s := NewReflectStrategy()

	cases := []struct {
		name     string
		val      any
		cfg      apis.Config
		expected string
	}{
		{"plain struct", A{}, cfg(), "strategy.A"},
		{"ptr", &A{}, cfg(), "strategy.A"},
		{"slice", []A{}, cfg(), "strategy.A"},
		{"array", [2]A{}, cfg(), "strategy.A"},
		{"chan", make(chan A), cfg(), "strategy.A"},
		{"map prefer elem (default)", map[string]A{}, cfg(), "strategy.A"},
		{"map prefer key (builtin visible)", map[string]A{}, cfg(func(c *apis.Config) {
			c.MapPreferElem = false
			c.IncludeBuiltins = true
		}), "string"},
		{"map prefer key (builtin hidden)", map[string]A{}, cfg(func(c *apis.Config) {
			c.MapPreferElem = false
			c.IncludeBuiltins = false
		}), ""},
		{"builtin visible", 42, cfg(func(c *apis.Config) { c.IncludeBuiltins = true }), "int"},
		{"builtin hidden", 42, cfg(func(c *apis.Config) { c.IncludeBuiltins = false }), ""},
		{"generic strips params", G[int]{}, cfg(), "strategy.G"},
		{"wrapped generic", []W[G[int]]{}, cfg(), "strategy.W"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := s.TryResolve(tc.val, tc.cfg)
			if tc.val == nil {
				if ok {
					t.Fatalf("nil value: expected ok=false, got true")
				}
				return
			}
			if !ok {
				t.Fatalf("expected ok=true for %T", tc.val)
			}
			if got != tc.expected {
				t.Fatalf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestReflectStrategy_ByType(t *testing.T) {
	s := NewReflectStrategy()

	cases := []struct {
		name     string
		typ      reflect.Type
		cfg      apis.Config
		expected string
	}{
		{"type plain", reflect.TypeOf(A{}), cfg(), "strategy.A"},
		{"type ptr", reflect.TypeOf(&A{}), cfg(), "strategy.A"},
		{"type slice", reflect.TypeOf([]A{}), cfg(), "strategy.A"},
		{"type array", reflect.TypeOf([2]A{}), cfg(), "strategy.A"},
		{"type chan", reflect.TypeOf((chan A)(nil)), cfg(), "strategy.A"},
		{"type map prefer elem", reflect.TypeOf(map[string]A{}), cfg(), "strategy.A"},
		{"type map prefer key visible", reflect.TypeOf(map[string]A{}), cfg(func(c *apis.Config) {
			c.MapPreferElem = false
			c.IncludeBuiltins = true
		}), "string"},
		{"type map prefer key hidden", reflect.TypeOf(map[string]A{}), cfg(func(c *apis.Config) {
			c.MapPreferElem = false
			c.IncludeBuiltins = false
		}), ""},
		{"type generic instantiation", reflect.TypeOf(G[int]{}), cfg(), "strategy.G"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := s.TryResolveType(tc.typ, tc.cfg)
			if tc.typ == nil {
				if ok {
					t.Fatalf("nil type: expected ok=false, got true")
				}
				return
			}
			if !ok {
				t.Fatalf("expected ok=true for %v", tc.typ)
			}
			if got != tc.expected {
				t.Fatalf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestReflectStrategy_MaxUnwrap(t *testing.T) {
	s := NewReflectStrategy()

	type PP = **A
	tt := reflect.TypeOf((*PP)(nil)).Elem() // **A type (not a value)

	// Too small MaxUnwrap -> cannot reach the named A.
	t.Run("tight limit", func(t *testing.T) {
		cfgTight := cfg(func(c *apis.Config) { c.MaxUnwrap = 1 })
		got, ok := s.TryResolveType(tt, cfgTight)
		if ok && got != "" {
			t.Fatalf("MaxUnwrap=1: expected empty/failed resolution, got %q", got)
		}
	})

	// Large enough -> success.
	t.Run("wide limit", func(t *testing.T) {
		cfgWide := cfg(func(c *apis.Config) { c.MaxUnwrap = 8 })
		got, ok := s.TryResolveType(tt, cfgWide)
		if !ok || got != "strategy.A" {
			t.Fatalf("MaxUnwrap=8: got (%q,%v), want (strategy.A,true)", got, ok)
		}
	})
}

// This test stresses the memoization and Normalize path under concurrency.
func TestReflectStrategy_Concurrent(t *testing.T) {
	s := NewReflectStrategy()
	conf := cfg()

	types := []reflect.Type{
		reflect.TypeOf(A{}),
		reflect.TypeOf(&A{}),
		reflect.TypeOf([]A{}),
		reflect.TypeOf(map[string]A{}),
		reflect.TypeOf(G[int]{}),
		reflect.TypeOf(W[G[int]]{}),
		reflect.TypeOf(0),
	}
	expect := []string{"strategy.A", "strategy.A", "strategy.A", "strategy.A", "strategy.G", "strategy.W", "int"}

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
				if !ok || got != expect[idx] {
					errCh <- got
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Fatalf("concurrent resolve mismatch: got=%q", e)
	}
}

// ---- Benchmarks ----

func BenchmarkReflectStrategy_ByType(b *testing.B) {
	s := NewReflectStrategy()

	types := []reflect.Type{
		reflect.TypeOf(A{}),
		reflect.TypeOf(&A{}),
		reflect.TypeOf([]A{}),
		reflect.TypeOf(map[string]A{}),
		reflect.TypeOf(G[int]{}),
		reflect.TypeOf(W[G[int]]{}),
		reflect.TypeOf(0),
	}

	configs := []struct {
		name string
		cfg  apis.Config
	}{
		{"default", cfg()},
		{"hide_builtins", cfg(func(c *apis.Config) { c.IncludeBuiltins = false })},
		{"prefer_key", cfg(func(c *apis.Config) { c.MapPreferElem = false })},
		{"low_maxunwrap", cfg(func(c *apis.Config) { c.MaxUnwrap = 1 })},
	}

	for _, cc := range configs {
		b.Run(cc.name, func(b *testing.B) {
			// Warm-up cache
			for _, t0 := range types {
				s.TryResolveType(t0, cc.cfg)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				t0 := types[i%len(types)]
				s.TryResolveType(t0, cc.cfg)
			}
		})
	}
}

func BenchmarkReflectStrategy_ByValue(b *testing.B) {
	s := NewReflectStrategy()

	values := []any{
		A{},
		&A{},
		[]A{},
		map[string]A{},
		G[int]{},
		W[G[int]]{},
		0,
	}

	conf := cfg()
	// Warm-up
	for _, v := range values {
		s.TryResolve(v, conf)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := values[i%len(values)]
		s.TryResolve(v, conf)
	}
}
