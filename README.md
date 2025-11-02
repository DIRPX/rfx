# rfx — Global Type-to-Name Resolver

`rfx` is a lightweight, reflection-aware identity resolution layer for Go.  
It provides a process-wide mechanism for mapping Go values and types to **stable, human-readable names** — suitable for logs, metrics, audit trails, and policy engines.

> **Goal:** unify how DIRPX components identify and describe entities across distributed binaries.

---

## ✳️ Motivation

Distributed systems often need a consistent way to **name** objects:
- `"authn.jwt"`, `"cache.entry"`, `"routing.policy"`, `"node.metrics"`, etc.  
Hard-coded strings are brittle and inconsistent; `reflect.TypeOf` is unstable across packages;  
and embedding names in structs scatters logic everywhere.

`rfx` solves this by providing a single, concurrent-safe, reflection-driven registry and resolver with clear semantics and no runtime locks on reads.

---

## 🧩 Design Overview

At runtime, `rfx` holds a **global immutable snapshot** (`state`) published atomically:

| Component    | Responsibility                                                       | Typical Implementation |
|--------------|----------------------------------------------------------------------|------------------------|
| **Config**   | Normalization rules for names (unwrap depth, builtin handling, etc.) | `apis.Config`          |
| **Registry** | Explicit map of Go types → string names                              | `apis.Registry`        |
| **Resolver** | Strategy chain that computes a name from a value or type             | `apis.Resolver`        |
| **Builder**  | Factory that constructs a new Registry and Resolver                  | `apis.Builder`         |
| **Strategy** | Strategy logic for Resolver chain                                    | `apis.Strategy`        |
| **Ext**      | Opaque extension payload for custom builders                         | `any`                  |

Reads use `atomic.Pointer` → zero locks.  
Writes (`SetConfig`, `SetBuilder`, etc.) build a new snapshot → atomic swap.

---

## ⚙️ Resolution Order

When resolving a name, the global `Resolver` applies strategies in order:

1. **Namer interface** — if a value implements `apis.Namer`, use its `EntityName()` method.  
   ```go
   type Namer interface { EntityName() string }
   ```
2. **Registry lookup** — if the type is explicitly registered, use that name.
3. **Reflect fallback** — derive `"pkg.Type"` name using normalization rules.

This guarantees a stable, canonical string for any Go type or value.

---

## 🧠 Concurrency Model

- **Reads are lock-free.**  
  `Entity()`, `EntityType()`, `Registry()`, and `Resolver()` simply load the current state atomically.

- **Writes are serialized.**  
  Updates (`SetConfig`, `SetBuilder`, `SetExt`, etc.) take a short build mutex, construct new layers if needed, and publish a new snapshot via atomic swap.

- **Pinning:**  
  - `SetRegistry()` and `SetResolver()` *pin* their respective layers to prevent rebuilds.  
  - Use `UnpinRegistry()` / `UnpinResolver()` to make them mutable again.

---

## 🔧 Public API (summary)

```go
// Read helpers
rfx.Entity(v any) string
rfx.EntityType(t reflect.Type) string
rfx.Registry() apis.Registry
rfx.Resolver() apis.Resolver

// Mutation helpers
rfx.SetConfig(cfg apis.Config)
rfx.SetBuilder(b apis.Builder)
rfx.SetExt(ext any)
rfx.SetRegistry(reg apis.Registry)
rfx.SetResolver(res apis.Resolver)
rfx.UnpinRegistry()
rfx.UnpinResolver()
rfx.SetAll(cfg *apis.Config, ext any, reg apis.Registry, res apis.Resolver, b apis.Builder)

// Extension helpers
rfx.ExtAs[T]() (T, bool)
```

All setters rebuild **only unpinned layers** using the current `Builder` and `Ext`.  
`SetAll` acts as a **hard reset**, ignoring pins — mostly used in tests.

---

## 🧩 Example

```go
package main

import (
	"fmt"
	"reflect"
	"dirpx.dev/rfx"
)

type MyType struct{}

// Custom naming via interface
func (MyType) EntityName() string { return "custom.type" }

func main() {
	// Register a type manually (optional)
	rfx.Registry().Register(reflect.TypeOf(MyType{}), "mytype.registered")

	// Simple resolution
	fmt.Println(rfx.Entity(MyType{}))          // → "custom.type"
	fmt.Println(rfx.EntityType(reflect.TypeOf(MyType{}))) // → "mytype.registered"

	// Change config at runtime (rebuilds resolver & registry)
	cfg := rfx.Registry().Entries() // diagnostic
	fmt.Println("registered entries:", len(cfg))
}
```

---

## 🧱 Builder Responsibilities

`Builder` defines how registries and resolvers are (re)built:

```go
type Builder interface {
    BuildRegistry(cfg Config, prev Registry, ext any) Registry
    BuildResolver(cfg Config, reg Registry, prev Resolver, ext any) Resolver
}
```

- `cfg` — current configuration (always normalized via `config.New()`).
- `prev` — previous instance (may be reused or migrated).
- `ext` — user-defined extension payload.
- `rfx`’s default builder composes three strategies:
  1. `NamerStrategy`
  2. `RegistryStrategy`
  3. `ReflectStrategy`

You can inject your own builder to alter resolution rules globally.

---

## 🧩 Ext Configuration

`ext` is an **opaque context** stored in the global snapshot.  
It is **not** interpreted by `rfx`; it is simply passed to the builder on every rebuild.

Typical uses:
- Custom naming policies (e.g., prefixing names per subsystem)
- Tenant or environment identifiers
- Context for plugins that hook into registry/resolver construction

```go
type MyExt struct {
    Prefix string
}

rfx.SetExt(MyExt{Prefix: "tenant-a"})
```

Your custom `Builder` can then downcast `ext.(MyExt)` and apply the prefix.

---

## 🧵 Concurrency Example

```go
// Safe concurrent reads
go func() {
    for {
        _ = rfx.EntityType(reflect.TypeOf(MyType{}))
    }
}()

// Configuration reload
go func() {
    for {
        rfx.SetConfig(newConfig())
    }
}()
```

All readers observe a consistent snapshot; no races or partial updates.

---

## 🧪 Testing Utilities

For deterministic tests:

```go
// Replace everything in one call
rfx.SetAll(&testCfg, nil, nil, nil, builder.New())

// Inject mock builder or registry
rfx.SetBuilder(mockBuilder)
rfx.SetRegistry(mockRegistry)
rfx.SetResolver(mockResolver)
rfx.UnpinRegistry()
rfx.UnpinResolver()
```

Use `rfx.ExtAs[T]()` to fetch custom extension values inside tests.

---

## 🧩 Internal Snapshot Lifecycle

```
+-------------------------+
|       state struct      |
|-------------------------|
| Config   → apis.Config  |
| Ext      → any          |
| Registry → apis.Registry|
| Resolver → apis.Resolver|
| Builder  → apis.Builder |
| Pin flags (reg/res)     |
+-------------------------+

            │
       atomic.Store
            ▼
    +-----------------+
    |  st *state ptr  | ← global snapshot
    +-----------------+
```

Each `Set*()` call creates a new `state`, fills it with correct layers,  
and swaps it in atomically. Old snapshots remain valid until GC.

---

## 🧱 Why It Matters

DIRPX uses `rfx` as the backbone for type-identity across all sub-systems:

- **dirpx-node**: labeling requests, plugin responses, cache entities.  
- **dirpx-cp**: consistent audit logs and metrics for policies and routing.  
- **dirpx-collector**: stable key generation for tracing and retention systems.  

It ensures that no matter where a struct originates, it has the same canonical name everywhere.

---

## 📜 License

Apache 2.0  
Copyright © 2025 DIRPX Authors.

---

## 🧭 Summary

| Trait | Description                                                  |
|-------|--------------------------------------------------------------|
| **Thread safety** | Lock-free reads, atomic snapshot swaps                       |
| **Extensibility** | Custom builders, custom extensions                           |
| **Determinism** | Stable `"pkg.Type"` format for all types                     |
| **Scope** | Naming only — not DI, not service discovery                  |
| **Integration** | Used in all DIRPX binaries (`node`, `cp`, `collector`, etc.) |

---

> “If everything has a name, nothing is anonymous — and that’s what makes distributed debugging possible.”
>
> — DIRPX Core Philosophy
