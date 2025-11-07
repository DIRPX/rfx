# rfx — Global Type-to-Name Resolver for Go

`rfx` is a lightweight, reflection‑aware identity layer that maps Go **types/values → stable names**.
Use it to get canonical, human‑readable identifiers for logging, metrics, policies, audit trails,
and cross‑process contracts — consistently, everywhere.

> **Goal:** one source of truth for entity names across all DIRPX components and services.

---

## Table of Contents

- [Why rfx](#why-rfx)
- [Key Ideas](#key-ideas)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Concepts & Architecture](#concepts--architecture)
  - [Registry](#registry)
  - [Resolver](#resolver)
  - [Builder](#builder)
  - [Config](#config)
  - [Snapshot & Pins](#snapshot--pins)
- [Resolution Order](#resolution-order)
- [Usage Patterns](#usage-patterns)
  - [Manual Registration](#manual-registration)
  - [Custom Names via Interface](#custom-names-via-interface)
  - [Pure Reflection Fallback](#pure-reflection-fallback)
  - [Bulk Diagnostics](#bulk-diagnostics)
- [Concurrency Model](#concurrency-model)
- [API Reference](#api-reference)
- [Advanced Topics](#advanced-topics)
  - [Custom Builder](#custom-builder)
  - [Using Extensions (`Ext`)](#using-extensions-ext)
  - [Testing & Determinism](#testing--determinism)
  - [Performance Notes](#performance-notes)
- [Examples](#examples)
- [Troubleshooting & FAQ](#troubleshooting--faq)
- [Design Principles](#design-principles)
- [Project Status & Versioning](#project-status--versioning)
- [License](#license)

---

## Why rfx

Distributed systems need consistent **names** for things: `authn.jwt`, `cache.entry`, `routing.policy`,
`node.metrics`, etc. String literals are brittle; `reflect.TypeOf` names vary with packages and build graph;
embedding ad‑hoc names in structs scatters logic across the codebase.

`rfx` centralizes naming:
- a single **registry** for explicit mappings,
- a pluggable **resolver** chain (interface → registry → reflection),
- an atomic **snapshot** published process‑wide for lock‑free reads.

The result is predictable, debuggable, and fast.

---

## Key Ideas

- **Stable identity**: the same type/value resolves to the same canonical string every time.
- **Zero locks on reads**: resolution is a simple atomic load + pure functions.
- **Replaceable strategy**: swap a `Builder` to change the resolution policy globally.
- **Minimal surface**: only naming — _not_ DI, _not_ wire codecs, _not_ discovery.
- **Test‑friendly**: whole‑system reset in a single call (`SetAll`), mockable interfaces.

---

## Installation

```bash
go get dirpx.dev/rfx@latest
```

> `rfx` is a standalone package. It depends only on the Go standard library.

---

## Quick Start

```go
package main

import (
	"fmt"
	"reflect"

	"dirpx.dev/rfx"
)

type Order struct{}
func (Order) EntityName() string { return "sales.order" } // optional interface

func main() {
	// 1) Optional explicit registration
	rfx.Registry().Register(reflect.TypeOf(Order{}), "sales.order.type")

	// 2) Resolve by value (uses EntityName() if present)
	fmt.Println(rfx.Entity(Order{})) // "sales.order"

	// 3) Resolve by reflect.Type (uses registry or fallback)
	fmt.Println(rfx.EntityType(reflect.TypeOf(Order{}))) // "sales.order.type" (from registry)

	// 4) Introspect current registry
	entries := rfx.Registry().Entries()
	fmt.Println("registered entries:", len(entries))
}
```

---

## Concepts & Architecture

At runtime, `rfx` holds a **global immutable snapshot** with five pieces:

```
+------------------------------+
| Config  | Ext | Registry | Resolver | Builder | PinFlags |
+------------------------------+
               │
        atomic.Store(*state)
               ▼
     process-global pointer
```

Updates rebuild missing layers and atomically swap the snapshot; readers always see a consistent view.

### Registry

Explicit mapping **Go type → name**; typically used for core types that must have fixed identities.

Minimal interface (illustrative):
```go
type Registry interface {
	Register(t reflect.Type, name string)
	Lookup(t reflect.Type) (name string, ok bool)
	Entries() map[reflect.Type]string // diagnostic copy
}
```

### Resolver

Computes a name from a value or type using a chain of **strategies** (see [Resolution Order](#resolution-order)).

### Builder

Factory that constructs `Registry` and `Resolver` when the snapshot needs rebuilding:

```go
type Builder interface {
	BuildRegistry(cfg Config, prev Registry, ext any) Registry
	BuildResolver(cfg Config, reg Registry, prev Resolver, ext any) Resolver
}
```

Switch builders to change global policy.

### Config

Normalization and policy flags for the resolver (e.g., how to unwrap pointers, whether to include package path, etc.).
Provided by `rfx.SetConfig` and stored in the snapshot.

### Snapshot & Pins

- `SetRegistry` / `SetResolver` **pin** the provided instances (they won’t be rebuilt automatically).
- `UnpinRegistry` / `UnpinResolver` clear those pins.
- `SetAll` resets everything in one go — great for tests.

---

## Resolution Order

The resolver tries, in order:

1. **Namer interface** — if a value implements:
   ```go
   type Namer interface { EntityName() string }
   ```
   then use that string.

2. **Registry lookup** — if the (possibly unwrapped) `reflect.Type` is registered, use it.

3. **Reflection fallback** — derive `"pkg.Type"` using `Config` rules (stable canonicalization).

This makes the default behavior sensible while keeping escape hatches obvious.

---

## Usage Patterns

### Manual Registration

```go
t := reflect.TypeOf((*MyInterface)(nil)).Elem()
rfx.Registry().Register(t, "feature.my_interface")
```

### Custom Names via Interface

```go
type Job struct { Kind string }
func (j Job) EntityName() string { return "batch.job." + j.Kind }
fmt.Println(rfx.Entity(Job{Kind: "reindex"})) // "batch.job.reindex"
```

### Pure Reflection Fallback

If you do nothing, `rfx` will derive a stable `"pkg.Type"` name. This is useful for quick diagnostics and when you don’t want to commit to explicit names yet.

### Bulk Diagnostics

```go
for t, name := range rfx.Registry().Entries() {
	fmt.Printf("%s => %s\n", t.String(), name)
}
```

---

## Concurrency Model

- **Reads:** lock‑free (`atomic.Pointer` load + pure resolution). Safe under heavy parallelism.
- **Writes:** short critical section to build a new snapshot; atomic swap publishes it.
- **No tearing:** readers never see partially applied updates.

Example:
```go
// Readers
go func() {
	for i := 0; i < 1_000_000; i++ {
		_ = rfx.EntityType(reflect.TypeOf(Order{}))
	}
}()

// Occasional config reload
go func() {
	rfx.SetConfig(loadConfigFromFile())
}()
```

---

## API Reference

High‑level helpers:
```go
// Resolution
func Entity(v any) string
func EntityType(t reflect.Type) string

// Introspection
func Registry() Registry
func Resolver() Resolver

// Snapshot mutation
func SetConfig(cfg Config)
func SetBuilder(b Builder)
func SetExt(ext any)
func SetRegistry(reg Registry)
func SetResolver(res Resolver)
func UnpinRegistry()
func UnpinResolver()
func SetAll(cfg *Config, ext any, reg Registry, res Resolver, b Builder)

// Extensions
func ExtAs[T any]() (T, bool)
```

> Functions above delegate to the current snapshot. Only unpinned layers are rebuilt when you call a `Set*` method.

---

## Advanced Topics

### Custom Builder

Provide a different strategy chain or registry implementation:

```go
type MyBuilder struct{}

func (MyBuilder) BuildRegistry(cfg rfx.Config, prev rfx.Registry, ext any) rfx.Registry {
	// reuse prev or wrap it with additional behavior
	return prev
}
func (MyBuilder) BuildResolver(cfg rfx.Config, reg rfx.Registry, prev rfx.Resolver, ext any) rfx.Resolver {
	// compose strategies in a custom order
	return NewMyResolverChain(reg, cfg)
}

rfx.SetBuilder(MyBuilder{})
```

### Using Extensions (`Ext`)

Opaque payload carried in the snapshot — accessible to your `Builder`:

```go
type NamingExt struct{ Prefix string }
rfx.SetExt(NamingExt{Prefix: "tenant-a"})
// Builder can downcast and prepend Prefix to all names it constructs.
```

Retrieve in tests or tooling:
```go
if ext, ok := rfx.ExtAs[NamingExt](); ok {
	fmt.Println(ext.Prefix)
}
```

### Testing & Determinism

```go
// One‑shot reset for isolation:
rfx.SetAll(&cfg, nil, nil, nil, mybuilder)

// Pin a mock registry:
rfx.SetRegistry(mockReg)   // pinned
rfx.UnpinRegistry()        // allow rebuild again
```

### Performance Notes

- Resolution avoids allocations on the hot path.
- No locks for readers; updates are rare and cheap.
- Fallback naming uses memoized reflection where appropriate (implementation detail).

---

## Examples

#### 1) Priorities in action
```go
type A struct{}
func (A) EntityName() string { return "custom.A" }

rfx.Registry().Register(reflect.TypeOf(A{}), "registered.A")

fmt.Println(rfx.Entity(A{}))                      // "custom.A"     (Namer wins)
fmt.Println(rfx.EntityType(reflect.TypeOf(A{})))  // "registered.A" (Registry)
```

#### 2) Swapping policy at runtime
```go
// switch to a builder that always prefers registry over interface
rfx.SetBuilder(builder.RegistryFirst{})
```

#### 3) Safe default without setup
```go
type X struct{}
fmt.Println(rfx.Entity(X{})) // e.g. "your/module/path.X"
```

---

## Troubleshooting & FAQ

**I’m getting different names across binaries. Why?**  
Make sure all processes register the same types, and that the module path (used by reflection) is consistent. Prefer explicit registration for cross‑binary stability.

**Do I need to register every type?**  
No. Register only important public entities. Others can rely on the fallback or the `Namer` interface.

**What about generics, pointers, aliases?**  
`Config` controls unwrapping and formatting rules. The default behaves sensibly, but you can swap it via `SetConfig` or customize a `Builder` to enforce your own policy.

**Is this a DI container?**  
No. `rfx` only resolves names; it does not construct values or manage lifecycles.

---

## Design Principles

- **Single responsibility:** naming only, nothing more.
- **Immutability by default:** publish immutable snapshots; never mutate in place.
- **Open for extension:** pluggable builder and strategies.
- **Predictable under load:** lock‑free reads, deterministic behavior.

---

## Project Status & Versioning

- Status: **early but stable for internal use**. API surface is intentionally small.
- Semver: breaking changes bump the major version. See releases for notes.
