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

// Package rfx provides a global, process-wide identity resolution service.
//
// rfx is responsible for turning "some Go value or type" into a stable,
// human-readable, domain-level name. This name can be used for logging,
// metrics, audit trails, policy engines, etc. Examples: "authn.jwt",
// "cache.entry", "routing.policy", "node.metrics", etc.
//
// # Design
//
// The core of rfx is a read-mostly global snapshot (state). The snapshot
// holds four things:
//
//   - Config: rules that control how types are normalized and named
//     (e.g. how deep to unwrap pointers/slices/maps, whether to allow
//     builtin types, etc.).
//
//   - Registry: a process-wide mapping from Go types to explicit,
//     human-chosen names. This is how you force stable names for important
//     domain entities like "ScoreRequest" or "RoutingPolicy". The registry
//     can be written to at runtime (Register).
//
//   - Resolver: a read-only object that answers "what is the name of this
//     value or type?". The resolver typically tries multiple strategies,
//     in priority order:
//     1. If the value implements apis.Namer, use v.EntityName().
//     2. If the type is found in the Registry, use that name.
//     3. Otherwise, fall back to a reflect-based strategy that derives
//     a stable "pkg.Type" identifier from the Go type.
//     Resolver is expected to be concurrency-safe for reads.
//
//   - Builder: a pluggable factory that knows how to construct Registry
//     and Resolver instances for a given Config (and optional extension
//     data). The Builder is also allowed to reuse/migrate state from
//     previous Registry/Resolver instances.
//
// All of these live inside a single immutable struct called state.
// The package holds an atomic pointer to the current state. Readers load
// that pointer, use it, and never mutate it. Writers build a brand-new
// state and atomically swap it in.
//
// This means rfx lookups are lock-free on the hot path:
//
//	name := rfx.Entity(obj)
//	kind := rfx.EntityType(reflect.TypeOf(obj))
//
// and concurrent callers always see a consistent snapshot.
//
// # Global API
//
// The package exposes three groups of operations:
//
//  1. Read helpers:
//
//     Entity(v any) string
//     EntityType(t reflect.Type) string
//     Registry() apis.Registry
//     Resolver() apis.Resolver
//
//     These are safe for concurrent use without additional locking.
//     They always read from the latest published snapshot.
//
//  2. Mutation helpers:
//
//     SetConfig(cfg apis.Config)
//     SetBuilder(b apis.Builder)
//     SetExt(ext T)
//     SetRegistry(reg apis.Registry)
//     SetResolver(res apis.Resolver)
//     UnpinRegistry()
//     UnpinResolver()
//     SetAll(...)
//
//     Each of these acquires an internal build lock, derives a new
//     snapshot (rebuilding or reusing Registry / Resolver as needed),
//     and then atomically publishes that snapshot.
//
//     Semantics in short:
//
//     - Config affects how names are computed (normalization rules).
//     Calling SetConfig() may trigger a rebuild of Registry and/or
//     Resolver, unless they are explicitly "pinned".
//
//     - Builder controls how Registry and Resolver are constructed.
//     Swapping the Builder lets you replace resolution logic
//     (different strategies, different naming policies) at runtime.
//
//     - Ext is an opaque extension payload. It is not interpreted by
//     rfx itself. It is simply passed down to the Builder so custom
//     builders (in other binaries) can carry extra policy/state.
//
//     - SetRegistry() / SetResolver() directly overwrite the current
//     Registry / Resolver in the snapshot and "pin" them. Once a
//     layer is pinned, rfx will stop rebuilding that layer
//     automatically until you call UnpinRegistry()/UnpinResolver().
//
//     - SetAll(...) is the "hard reset" API. It lets a process replace
//     Builder, Config, Ext, Registry, Resolver in one shot. This is
//     mainly used by tests to get a clean deterministic state
//     between test cases.
//
//  3. Introspection:
//
//     ExtAs[T]() (T, bool)
//     // plus Registry().Entries(), etc.
//
//     These let callers examine the currently published snapshot for
//     debugging, metrics exposition, or documentation.
//
// # Concurrency model
//
// Reads (Entity, EntityType, Registry, Resolver) are wait-free: they load
// the current *state atomically and never take locks. The Resolver and
// Registry returned by that state must themselves be concurrency-safe
// for reads.
//
// Writes (SetConfig, SetBuilder, SetExt, SetRegistry, SetResolver, etc.)
// take a short build mutex, assemble a brand-new state struct, and then
// publish it via an atomic pointer swap. This gives the calling binary
// a predictable "last write wins" behavior without forcing per-lookup
// locking.
//
// # Pinning
//
// rfx supports the concept of "pinning" a layer:
//
//   - When you call SetRegistry(reg), that exact Registry becomes the
//     process-wide registry and is considered pinned. Further calls to
//     SetConfig() will not attempt to rebuild a new Registry until you
//     explicitly UnpinRegistry().
//
//   - When you call SetResolver(res), that Resolver is pinned and will
//     not be rebuilt automatically until UnpinResolver().
//
// Pinning is there for advanced scenarios where you want full control
// over one layer while still letting other layers evolve. For example,
// you may lock a custom Resolver for audit/telemetry formatting but still
// allow the rest of the system to change Config values.
//
// # Extension config
//
// The snapshot also carries an "ext" field. This is an opaque interface{}
// (any) value owned by the embedding binary (for example, dirpx-node or
// dirpx-cp). rfx does not interpret ext. The active Builder receives ext
// on each rebuild, so out-of-tree builders can inject custom naming rules
// or policy logic without hacking the rfx core.
//
// # Usage pattern in a binary
//
// A typical component does:
//
//  1. Let rfx init with default builder/config.
//
//  2. Optionally call rfx.SetExt(myCustomPolicy) so the Builder can see
//     extra naming policy.
//
//  3. Optionally register well-known types up front:
//
//     rfx.Registry().Register(reflect.TypeOf(MyRequest{}), "request.myapi")
//     rfx.Registry().Register(reflect.TypeOf(MyError{}),   "error.myapi")
//
//  4. Use rfx.Entity(...) everywhere in logs, traces, metrics.
//
//  5. In tests, call rfx.SetAll(...) to get deterministic snapshots
//     and to inject a mock Builder.
//
// # Scope
//
// rfx is intentionally small. It does not try to be a general DI container
// or service locator. It only solves one job:
//
//	"Given any Go value or type, produce a stable, human-readable name
//	 that makes sense for operators, logs, metrics, and policy."
//
// Everything else (lifecycle, injection, authz, routing, etc.) belongs to
// higher layers.
package rfx
