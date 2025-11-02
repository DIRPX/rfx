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

package reflect

import (
	"errors"
	"reflect"

	"dirpx.dev/rfx/apis"
	"dirpx.dev/rfx/config"
)

var (
	// ErrReflectNilType is returned when a nil reflect.Type is provided.
	ErrReflectNilType = errors.New("reflect: nil reflect.Type provided")
	// ErrReflectTypeNotNamed indicates that the provided type (after unwrapping containers)
	// does not contain a named type (e.g., anonymous struct, func, interface{}).
	ErrReflectTypeNotNamed = errors.New("reflect: type has no registered name")
)

// Normalize unwraps containers according to config (MaxUnwrap/MapPreferElem)
// and returns the nearest named inner type, or an error if none is found.
//
// Unwrapping policy:
//   - ptr/slice/array/chan  -> Elem()
//   - map[K]V: try preferred side first (Elem if MapPreferElem; otherwise Key);
//     if the preferred side is named, return it;
//     else try the other side; if still unnamed, continue unwrapping Elem().
//   - default: if t.Name() != "", return t; otherwise ErrNotNamed.
//
// If MaxUnwrap <= 0, DefaultMaxUnwrap is used.
func Normalize(t reflect.Type, cfg apis.Config) (reflect.Type, error) {
	if t == nil {
		return nil, ErrReflectNilType
	}
	maxUnwrap := cfg.MaxUnwrap
	if maxUnwrap <= 0 {
		maxUnwrap = config.DefaultMaxUnwrap
	}

	preferElem := cfg.MapPreferElem

	for i := 0; t != nil && i < maxUnwrap; i++ {
		switch t.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Chan:
			t = t.Elem()

		case reflect.Map:
			// Try preferred side
			if preferElem {
				et := t.Elem()
				if et != nil && et.Name() != "" {
					return et, nil
				}
				// Fallback to the other side
				kt := t.Key()
				if kt != nil && kt.Name() != "" {
					return kt, nil
				}
				// Neither side named: keep unwrapping element
				t = et
			} else {
				kt := t.Key()
				if kt != nil && kt.Name() != "" {
					return kt, nil
				}
				et := t.Elem()
				if et != nil && et.Name() != "" {
					return et, nil
				}
				t = et
			}

		default:
			// Named, return; anonymous -> error
			if t.Name() != "" {
				return t, nil
			}
			return nil, ErrReflectTypeNotNamed
		}
	}

	// After reaching max depth, ensure we ended on a named type.
	if t != nil && t.Name() != "" {
		return t, nil
	}
	return nil, ErrReflectTypeNotNamed
}
