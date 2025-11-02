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

package reflect_test

import (
	"errors"
	"reflect"
	"runtime"
	"sync"
	"testing"

	"dirpx.dev/rfx/apis"
	uref "dirpx.dev/rfx/utils/reflect"
)

// Local test types.
type A struct{}
type B struct{}
type G[T any] struct{}
type W[T any] struct{ V T }

// cfg returns a convenient baseline Config for tests.
func cfg(opts ...func(*apis.Config)) apis.Config {
	c := apis.Config{
		IncludeBuiltins: true, // unused by Normalize itself, harmless to pass
		MaxUnwrap:       8,
		MapPreferElem:   true,
	}
	for _, o := range opts {
		o(&c)
	}
	return c
}

func TestNormalize_BasicContainers(t *testing.T) {
	conf := cfg()

	cases := []struct {
		name string
		typ  reflect.Type
		want reflect.Type
	}{
		{"plain", reflect.TypeOf(A{}), reflect.TypeOf(A{})},
		{"ptr", reflect.TypeOf(&A{}), reflect.TypeOf(A{})},
		{"slice", reflect.TypeOf([]A{}), reflect.TypeOf(A{})},
		{"array", reflect.TypeOf([2]A{}), reflect.TypeOf(A{})},
		{"chan", reflect.TypeOf((chan A)(nil)), reflect.TypeOf(A{})},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := uref.Normalize(tc.typ, conf)
			if err != nil {
				t.Fatalf("Normalize(%v) returned error: %v", tc.typ, err)
			}
			if got != tc.want {
				t.Fatalf("Normalize(%v) = %v, want %v", tc.typ, got, tc.want)
			}
		})
	}
}

func TestNormalize_MapPreference(t *testing.T) {
	// map[string]A: elem is A (named), key is string (builtin named)
	tMap := reflect.TypeOf(map[string]A{})

	// Prefer element (default) -> A
	got1, err1 := uref.Normalize(tMap, cfg())
	if err1 != nil {
		t.Fatalf("Normalize(map[string]A) prefer elem: %v", err1)
	}
	if got1 != reflect.TypeOf(A{}) {
		t.Fatalf("prefer elem: got %v, want A", got1)
	}

	// Prefer key -> string
	got2, err2 := uref.Normalize(tMap, cfg(func(c *apis.Config) { c.MapPreferElem = false }))
	if err2 != nil {
		t.Fatalf("Normalize(map[string]A) prefer key: %v", err2)
	}
	if got2 != reflect.TypeOf("") {
		t.Fatalf("prefer key: got %v, want string", got2)
	}
}

func TestNormalize_GenericInstantiation(t *testing.T) {
	conf := cfg()

	gt, err := uref.Normalize(reflect.TypeOf(G[int]{}), conf)
	if err != nil {
		t.Fatalf("Normalize(G[int]{}): %v", err)
	}
	// Implementation detail of Type.Name() may keep params; ensure the type is named.
	if gt == nil || gt.Name() == "" {
		t.Fatalf("Normalize(G[int]{}) returned unnamed or nil type: %v", gt)
	}

	wt, err := uref.Normalize(reflect.TypeOf(W[G[int]]{}), conf)
	if err != nil {
		t.Fatalf("Normalize(W[G[int]]{}): %v", err)
	}
	if wt == nil || wt.Name() == "" {
		t.Fatalf("Normalize(W[G[int]]{}) returned unnamed or nil type: %v", wt)
	}
}

func TestNormalize_MaxUnwrap(t *testing.T) {
	// **A with low MaxUnwrap should fail, with larger MaxUnwrap should succeed.
	type PP = **A
	tPP := reflect.TypeOf((*PP)(nil)).Elem() // the **A type itself

	// Tight limit -> expect an error.
	if _, err := uref.Normalize(tPP, cfg(func(c *apis.Config) { c.MaxUnwrap = 1 })); err == nil {
		t.Fatalf("MaxUnwrap=1: expected error, got nil")
	}

	// Wide limit -> expect success.
	if got, err := uref.Normalize(tPP, cfg(func(c *apis.Config) { c.MaxUnwrap = 8 })); err != nil || got != reflect.TypeOf(A{}) {
		t.Fatalf("MaxUnwrap=8: got (%v,%v), want (A,nil)", got, err)
	}
}

func TestNormalize_Errors(t *testing.T) {
	// Nil type -> error.
	if _, err := uref.Normalize(nil, cfg()); err == nil {
		t.Fatalf("nil type: expected error, got nil")
	}

	// Anonymous struct -> error (no nearest named type).
	var anon = struct{ X int }{}
	if _, err := uref.Normalize(reflect.TypeOf(anon), cfg()); err == nil {
		t.Fatalf("anonymous struct: expected error, got nil")
	}
}

func TestNormalize_MapUnnamedElemFallback(t *testing.T) {
	// map[string]struct{X int}: prefer elem, but elem is anonymous; fallback should consider key.
	type Anon = struct{ X int }
	tMap := reflect.TypeOf(map[string]Anon{})

	got1, err1 := uref.Normalize(tMap, cfg(func(c *apis.Config) { c.MapPreferElem = true }))
	if err1 != nil {
		t.Fatalf("map[string]Anon prefer elem: %v", err1)
	}
	if got1 != reflect.TypeOf("") {
		t.Fatalf("prefer elem -> fallback to key: got %v, want string", got1)
	}

	// Prefer key => string
	got2, err2 := uref.Normalize(tMap, cfg(func(c *apis.Config) { c.MapPreferElem = false }))
	if err2 != nil {
		t.Fatalf("map[string]Anon prefer key: %v", err2)
	}
	if got2 != reflect.TypeOf("") {
		t.Fatalf("prefer key: got %v, want string", got2)
	}
}

// This test stresses Normalize concurrently to smoke-test thread safety of the logic
// (Normalize should be pure; no shared state is mutated here).
func TestNormalize_Concurrent(t *testing.T) {
	types := []reflect.Type{
		reflect.TypeOf(A{}),
		reflect.TypeOf(&A{}),
		reflect.TypeOf([]A{}),
		reflect.TypeOf(map[string]A{}),
		reflect.TypeOf(G[int]{}),
		reflect.TypeOf(W[G[int]]{}),
		reflect.TypeOf(0),
	}
	conf := cfg()

	workers := runtime.GOMAXPROCS(0) * 4
	iters := 2000

	var wg sync.WaitGroup
	wg.Add(workers)

	errCh := make(chan error, workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				tt := types[i%len(types)]
				rt, err := uref.Normalize(tt, conf)
				if err != nil {
					errCh <- err
					return
				}
				if rt == nil || rt.Name() == "" {
					errCh <- errors.New("got unnamed or nil type")
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Fatal(e)
	}
}

func BenchmarkNormalize_ByType(b *testing.B) {
	types := []reflect.Type{
		reflect.TypeOf(A{}),
		reflect.TypeOf(&A{}),
		reflect.TypeOf([]A{}),
		reflect.TypeOf(map[string]A{}),
		reflect.TypeOf(G[int]{}),
		reflect.TypeOf(W[G[int]]{}),
		reflect.TypeOf(0),
	}
	conf := cfg()

	// Warm-up to exercise paths.
	for _, t0 := range types {
		_, _ = uref.Normalize(t0, conf)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = uref.Normalize(types[i%len(types)], conf)
	}
}

func BenchmarkNormalize_VariousConfigs(b *testing.B) {
	tMap := reflect.TypeOf(map[string]A{})
	configs := []apis.Config{
		cfg(),
		cfg(func(c *apis.Config) { c.MapPreferElem = false }),
		cfg(func(c *apis.Config) { c.MaxUnwrap = 1 }),
	}

	for _, c := range configs {
		b.Run(runName(c), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = uref.Normalize(tMap, c)
			}
		})
	}
}

// runName builds a compact sub-benchmark name like "M-E-U8-B+" safely.
func runName(c apis.Config) string {
	// Map side: E/K; Builtins: +/-; Unwrap: U<number> (default to 8 if <= 0).
	m := byte('E')
	if !c.MapPreferElem {
		m = 'K'
	}
	b := byte('+')
	if !c.IncludeBuiltins {
		b = '-'
	}
	u := c.MaxUnwrap
	if u <= 0 {
		u = 8
	}

	// Assemble without allocations, with ample buffer.
	var buf [16]byte
	i := 0
	buf[i] = 'M'
	i++
	buf[i] = '-'
	i++
	buf[i] = m
	i++
	buf[i] = '-'
	i++
	buf[i] = 'U'
	i++

	// Write u as ASCII digits.
	var tmp [4]byte
	j := 0
	n := u
	for {
		tmp[j] = byte('0' + n%10)
		j++
		n /= 10
		if n == 0 {
			break
		}
	}
	for k := j - 1; k >= 0; k-- {
		buf[i] = tmp[k]
		i++
	}

	buf[i] = '-'
	i++
	buf[i] = 'B'
	i++
	buf[i] = b
	i++

	return string(buf[:i])
}
