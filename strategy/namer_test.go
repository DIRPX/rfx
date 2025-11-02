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
	"testing"

	"dirpx.dev/rfx/apis"
	"dirpx.dev/rfx/strategy"
)

type namedType struct{}

func (namedType) EntityName() string { return "custom.Name" } // implements rfx.Namer

func TestNamerStrategy_TryResolve(t *testing.T) {
	s := strategy.NewNamerStrategy()
	conf := apis.Config{} // config is irrelevant for NamerStrategy

	// With value implementing rfx.Namer -> handled = true
	got, ok := s.TryResolve(namedType{}, conf)
	if !ok || got != "custom.Name" {
		t.Fatalf("TryResolve: got (%q,%v), want (custom.Name,true)", got, ok)
	}

	// With non-namer value -> handled = false
	got, ok = s.TryResolve(struct{}{}, conf)
	if ok || got != "" {
		t.Fatalf("TryResolve(non-namer): got (%q,%v), want ('',false)", got, ok)
	}

	// TryResolveType should never handle (no instance)
	typ := reflect.TypeOf(namedType{})
	got, ok = s.TryResolveType(typ, conf)
	if ok || got != "" {
		t.Fatalf("TryResolveType: got (%q,%v), want ('',false)", got, ok)
	}
}

// Ensure the local type actually satisfies rfx.Namer (compile-time).
var _ apis.Namer = (*namedType)(nil)
