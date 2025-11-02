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

// Config carries read-only resolution knobs that influence strategies.
// It is passed by value and should be treated as immutable by implementations.
type Config struct {
	// IncludeBuiltins controls whether builtin/no-package named types
	// (e.g., "int", "string") are returned as names. If false, such cases yield "".
	IncludeBuiltins bool

	// MaxUnwrap limits container unwrapping depth (ptr/slice/array/chan/map).
	// Acts as a safety guard against pathological nesting.
	MaxUnwrap int

	// MapPreferElem controls which side of map[K]V is considered “primary”
	// when searching for a nearest named inner type. If true, prefer V; otherwise K.
	MapPreferElem bool
}
