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

package common

// Namer identifies application-level entities by a stable, canonical name.
//
// # Overview
//
// Namer is the primary, zero-reflection fast-path for resolving entity names
// inside the rfx reflection subsystem. When a value implements Namer, the
// resolution logic MUST prefer this interface and MUST NOT attempt any
// additional strategies (such as type-based naming, struct tags, or registry
// lookups) for that value.
//
// Semantically, Namer is a type-level contract: EntityName describes the
// *kind* of entity, not a particular instance. The returned name is expected
// to be independent of instance state and to remain stable across program
// executions, deployments, and process restarts, as long as the underlying
// domain model does not change.
//
// # Performance
//
// Implementations are intended to be extremely cheap:
//
//   - SHOULD be constant-time and amortized O(1).
//   - SHOULD avoid heap allocations on the hot path.
//   - MUST NOT perform blocking operations or I/O.
//   - MUST be safe to call from multiple goroutines concurrently.
//
// # Usage
//
// Typical usage is to define a small, domain-specific name that can be used
// for logging, metrics, tracing, routing, or registry lookups:
//
//	type User struct {
//	    ID   string
//	    Name string
//	}
//
//	func (User) EntityName() string {
//	    return "domain.user"
//	}
//
//	user := User{ID: "123"}
//	name := rfx.Entity(user) // Returns "domain.user" via Namer fast-path.
//
// # Naming guidelines
//
// In general, the EntityName value is expected to be:
//
//   - Stable across program executions (MUST).
//   - Unique within the application’s logical namespace (SHOULD).
//   - Short and human-readable (SHOULD; <64 characters RECOMMENDED).
//   - Expressed in a conventional format, such as lowercase,
//     dot-separated segments (MAY, but strongly RECOMMENDED).
type Namer interface {
	// EntityName returns the canonical, type-level name for this entity.
	//
	// # Contract
	//
	//   - The returned name MUST be non-empty.
	//   - The returned name MUST be deterministic for a given concrete type.
	//   - The returned name MUST NOT depend on mutable instance state
	//     (for example, field values that vary per object).
	//   - The implementation MUST be safe for concurrent calls from multiple
	//     goroutines.
	//
	// # Performance and side-effects
	//
	//   - Implementations SHOULD avoid heap allocations; returning a constant
	//     string literal or a precomputed value is RECOMMENDED.
	//   - Implementations MUST NOT perform blocking operations, system calls,
	//     or I/O.
	//   - Implementations MUST NOT perform expensive computations on the hot
	//     path; if a name needs to be derived, it SHOULD be precomputed and
	//     cached at type initialization time.
	//
	// # Semantics
	//
	// The returned value is intended to serve as a canonical identifier for
	// logging, metrics, tracing, routing, and internal registries.
	//
	// Callers MAY treat this name as stable across the lifetime of the
	// process, but they MUST NOT assume that different applications or
	// binaries use the same naming scheme unless explicitly coordinated.
	EntityName() string
}

// TypeNamer provides generic, type-aware naming for values of type T.
//
// # Overview
//
// TypeNamer is a generic, type-parametric naming interface. It allows
// different naming strategies to be expressed in terms of a Go type parameter
// `T`, while still producing canonical entity names that can be consumed by
// the rfx reflection subsystem, registries, loggers, or metrics backends.
//
// Unlike Namer, which is typically implemented as a method on the entity
// type itself, TypeNamer[T] separates:
//
//   - The *subject* being named (a value of type T), and
//   - The *strategy* that decides how to derive its name.
//
// This is useful when:
//
//   - The same naming strategy should be reused across multiple types.
//   - Name derivation needs to be configured or injected (for example,
//     per module, per subsystem, or per environment).
//   - You want to experiment with different naming policies without
//     changing the entity types.
//
// Implementations MAY inspect both the static type T and the dynamic type
// (when T is an interface), as well as selected aspects of the value v.
// However, for use as canonical entity identifiers, names SHOULD be primarily
// type-level and stable for a given concrete type.
//
// # Usage
//
// A typical pattern is to define a generic struct that implements TypeNamer
// for any T:
//
//   type CustomNamer[T any] struct{}
//
//   func (n CustomNamer[T]) EntityName(v T) string {
//       // Custom logic based on type T (and optionally v).
//       return fmt.Sprintf("custom.%T", v)
//   }
//
//   var namer TypeNamer[*User] = CustomNamer[*User]{}
//   name := namer.EntityName(&User{ID: "123"}) // e.g. "custom.*rfx.User"
//
// Implementations MAY be adapted to Namer or other naming abstractions by
// capturing a particular T and value category (for example, pointer vs value
// receivers) and exposing a zero-argument, type-level EntityName.
type TypeNamer[T any] interface {
	// EntityName returns a canonical name for a value of type T.
	//
	// # Contract
	//
	//   - The returned name MUST be a valid entity name according to the
	//     conventions used by the surrounding system (for example, the same
	//     rules that apply to Namer).
	//   - The returned name MUST be deterministic for a given input v.
	//   - For canonical, type-level naming, the result SHOULD depend only on
	//     the concrete type of v (including its dynamic type when T is an
	//     interface), not on its mutable instance state.
	//   - Implementations MUST be safe for concurrent calls from multiple
	//     goroutines.
	//
	// # Performance and side-effects
	//
	//   - Implementations SHOULD keep per-call cost low (ideally O(1) with
	//     respect to v), and SHOULD avoid unnecessary heap allocations.
	//   - Implementations MUST NOT perform blocking operations or I/O in
	//     EntityName.
	//   - If computing the name requires reflection or string building,
	//     implementations SHOULD precompute and cache reusable components
	//     (for example, a type-based prefix) where feasible.
	//
	// # Semantics
	//
	// The returned name is suitable for use as:
	//
	//   - A canonical identifier for logging and metrics.
	//   - A key in internal registries or caches.
	//   - A routing or dispatch hint in higher-level frameworks.
	//
	// Callers MAY assume that names produced by a given TypeNamer[T] are
	// consistent over the lifetime of the process, but they MUST NOT assume
	// cross-process or cross-application compatibility unless explicitly
	// documented by the implementation.
	EntityName(v T) string
}


// NamerFunc adapts a plain function to the Namer interface.
//
// # Overview
//
// NamerFunc is a convenience adapter that allows standalone functions with
// signature `func() string` to satisfy the Namer interface. This is useful
// when the entity name is naturally expressed as a function (for example,
// when it must be computed, or when you want to pass naming behavior as a
// dependency) rather than as a method on the entity type itself.
//
// Using NamerFunc does not change the semantics of Namer: the function is
// still expected to return a stable, type-level canonical name that does not
// depend on mutable instance state and remains stable across program
// executions as long as the domain model is unchanged.
//
// # Usage
//
//	func userEntityName() string { return "domain.user" }
//
//	var namer Namer = NamerFunc(userEntityName)
//	name := namer.EntityName() // "domain.user"
//
// # Contract
//
//   - A NamerFunc MUST return a non-empty, deterministic string.
//   - The returned name MUST be suitable as a canonical identifier for the
//     entity kind (type-level name, not per-instance).
//   - NamerFunc implementations MUST be safe to call from multiple goroutines
//     concurrently.
//   - NamerFunc SHOULD avoid heap allocations and expensive work on the hot
//     path, just like any other Namer implementation.
//   - NamerFunc MUST NOT perform blocking operations or I/O.
//
// # Performance
//
// NamerFunc adds virtually no overhead compared to calling the underlying
// function directly: EntityName is a single function call indirection with
// no additional allocations under normal circumstances.
type NamerFunc func() string

// EntityName implements Namer for NamerFunc.
//
// # Semantics
//
// Calling EntityName on a NamerFunc is equivalent to invoking the underlying
// function value directly. All contractual requirements of Namer apply to the
// wrapped function:
//
//   - It MUST return a non-empty, deterministic, type-level name.
//   - It MUST be safe for concurrent use by multiple goroutines.
//   - It MUST NOT perform blocking I/O or long-running computations on the
//     hot path.
//   - It SHOULD keep per-call overhead minimal (ideally constant-time, with
//     no heap allocations).
//
// # Notes
//
// If the underlying function performs caching or precomputation, that logic
// SHOULD be implemented in a concurrency-safe manner (for example, using
// package-level initialization or sync.Once) so that repeated calls to
// EntityName remain cheap and predictable.
func (f NamerFunc) EntityName() string {
	return f()
}
