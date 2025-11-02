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

package apis

import "reflect"

// Registry provides an optional reflection-free lookup for known types.
// Keep it minimal so implementations can be lock-free or sync.Map-backed.
type Registry interface {
	// Register associates a (nearest named) reflect.Type with a fixed name.
	// Implementations should be idempotent; conflicting re-registrations may panic.
	Register(t reflect.Type, name string) error
	// Lookup returns a name for a type if present.
	Lookup(t reflect.Type) (name string, ok bool)
	// Entries returns a snapshot for diagnostics/docs (order is unspecified).
	Entries() []Entry
	// Count returns the number of registered entries.
	Count() int
	// Reset clears all registered entries.
	Reset()
}

// Entry is a single (type, name) association in a Registry snapshot.
type Entry struct {
	// Type is the registered reflect.Type.
	Type reflect.Type
	// Name is the associated name.
	Name string
}
