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
	"path"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	apis "dirpx.dev/rfx/apis"
	"dirpx.dev/rfx/strategy"
)

// Named types for stable names.
type Foo struct{}
type Bar[T any] struct{ X T }

func mustNonEmpty(t *testing.T, s string) {
	t.Helper()
	if strings.TrimSpace(s) == "" {
		t.Fatal("expected non-empty name")
	}
}

// TestReflectStrategy_ConcurrentResolve_NoRace verifies that TryResolve/TryResolveType
// are race-free and return stable names under heavy concurrency.
func TestReflectStrategy_ConcurrentResolve_NoRace(t *testing.T) {
	s := strategy.NewReflectStrategy()
	cfg := apis.Config{
		IncludeBuiltins: true, // allow names for builtin types too
		MapPreferElem:   true,
		MaxUnwrap:       8,
	}

	vals := []any{
		Foo{}, &Foo{}, []Foo{}, [2]Foo{}, make(chan Foo),
		Bar[int]{}, &Bar[string]{},
		123, "abc", []byte{1, 2, 3}, map[string]int{"a": 1},
	}
	tys := []reflect.Type{
		reflect.TypeOf(Foo{}),
		reflect.TypeOf(&Foo{}),
		reflect.TypeOf([]Foo{}),
		reflect.TypeOf(map[string]int{}),
		reflect.TypeOf(Bar[int]{}),
	}

	// Single-thread sanity.
	for _, v := range vals {
		if name, ok := s.TryResolve(v, cfg); !ok {
			t.Fatalf("TryResolve failed for %T", v)
		} else {
			mustNonEmpty(t, name)
		}
	}
	for _, tt := range tys {
		if name, ok := s.TryResolveType(tt, cfg); !ok {
			t.Fatalf("TryResolveType failed for %v", tt)
		} else {
			mustNonEmpty(t, name)
		}
	}

	// Concurrent hammer.
	wg := sync.WaitGroup{}
	workers := runtime.GOMAXPROCS(0) * 4
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 5000; i++ {
				v := vals[(i+id)%len(vals)]
				if name, ok := s.TryResolve(v, cfg); !ok || name == "" {
					t.Errorf("TryResolve failed for %T (ok=%v, name=%q)", v, ok, name)
					return
				}
				tt := tys[(i+id)%len(tys)]
				if name, ok := s.TryResolveType(tt, cfg); !ok || name == "" {
					t.Errorf("TryResolveType failed for %v (ok=%v, name=%q)", tt, ok, name)
					return
				}
			}
		}(w)
	}
	wg.Wait()
}

// Optional: quick heuristic that package segment is present for non-builtin types.
func TestReflectStrategy_PackagePrefix_ForUserTypes(t *testing.T) {
	s := strategy.NewReflectStrategy()
	cfg := apis.Config{IncludeBuiltins: true, MapPreferElem: true, MaxUnwrap: 8}

	name, ok := s.TryResolve(Foo{}, cfg)
	if !ok {
		t.Fatal("TryResolve failed for Foo")
	}
	// Expect something like "<pkg>.<Type>", where pkg is the last element of PkgPath.
	pp := reflect.TypeOf(Foo{}).PkgPath()
	wantPkg := path.Base(pp)
	if !strings.HasPrefix(name, wantPkg+".") {
		t.Fatalf("unexpected name: %q (want prefix %q.)", name, wantPkg)
	}
}
