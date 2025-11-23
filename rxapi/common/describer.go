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

// Describer augments Namer with human-oriented metadata about an entity type.
//
// # Overview
//
// Describer is a higher-level contract that extends Namer with additional,
// human-readable metadata about an entity type. While Namer focuses on a
// compact, canonical identifier (for logging, metrics, and internal
// registries), Describer provides context that is useful for:
//
//   - Documentation and API browsers.
//   - Debugging and introspection tools.
//   - Administrative and developer-facing UIs.
//   - Schema evolution and compatibility checks.
//
// All methods on Describer are type-level: they describe the *kind* of
// entity, not any particular instance. Implementations SHOULD return values
// that are stable for a given version of the type’s schema and do not depend
// on mutable runtime state.
//
// # Usage
//
//	type User struct {
//	    ID   string
//	    Name string
//	}
//
//	func (User) EntityName() string        { return "domain.user" }
//	func (User) EntityDescription() string { return "User account in the system" }
//	func (User) EntityCategory() string    { return "identity" }
//	func (User) EntityVersion() string     { return "v1" }
//
// This metadata can then be consumed by higher-level frameworks to generate
// documentation, drive navigation, or display human-friendly descriptions
// alongside canonical identifiers.
//
// # Contract
//
//   - All methods MUST be safe for concurrent use by multiple goroutines.
//   - All methods SHOULD be inexpensive and ideally allocation-free on the
//     hot path (for example, returning string literals or precomputed values).
//   - Implementations MUST NOT perform blocking operations or I/O.
//   - Returned values SHOULD be deterministic for a given type and schema
//     version; changes SHOULD correspond to deliberate schema or behavior
//     changes rather than transient runtime conditions.
type Describer interface {
	Namer

	// EntityDescription returns a human-readable description of the entity type.
	//
	// # Semantics
	//
	// EntityDescription is intended to be a concise, human-oriented summary
	// of what the entity represents in the domain model. It is typically
	// used in:
	//
	//   - Documentation or schema browsers.
	//   - Admin consoles and configuration UIs.
	//   - Debugging tools and introspection views.
	//
	// Recommended properties:
	//
	//   - SHOULD be a short, single-sentence description.
	//   - SHOULD be stable for a given version of the entity schema.
	//   - SHOULD be understandable by humans without requiring knowledge
	//     of internal naming conventions.
	//
	// Localization:
	//
	//   - Implementations MAY return a description in a default locale
	//     (for example, English) if the system is not localization-aware.
	//   - If multiple locales are supported, higher-level infrastructure
	//     SHOULD handle locale selection; this interface models only the
	//     default, locale-agnostic description.
	//
	// # Contract
	//
	//   - The returned string MAY be empty if no description is available,
	//     but callers SHOULD handle that case gracefully (for example, by
	//     falling back to EntityName).
	//   - The implementation MUST be safe for concurrent calls and MUST NOT
	//     perform blocking I/O or long-running computations.
	EntityDescription() string

	// EntityCategory returns a coarse-grained category or domain for the entity type.
	//
	// # Semantics
	//
	// EntityCategory provides a high-level grouping that can be used for
	// organizing entities in UIs, documentation, or metrics dashboards. It
	// is typically drawn from a small, controlled vocabulary such as:
	//
	//   - "identity"
	//   - "catalog"
	//   - "payment"
	//   - "analytics"
	//   - "messaging"
	//
	// Recommended properties:
	//
	//   - SHOULD be relatively short (for example, a single word or slug).
	//   - SHOULD be stable across versions of the same entity type.
	//   - SHOULD come from an application-wide controlled set of categories
	//     to keep navigation and grouping consistent.
	//
	// # Contract
	//
	//   - The returned string MAY be empty if the entity does not belong
	//     to a well-defined category, but infrastructure SHOULD be prepared
	//     to handle that case (for example, by grouping under "uncategorized").
	//   - The implementation MUST be safe for concurrent calls and SHOULD
	//     avoid allocations on the hot path (for example, by returning a
	//     string literal or precomputed value).
	EntityCategory() string

	// EntityVersion returns a schema or contract version for the entity type.
	//
	// # Semantics
	//
	// EntityVersion is intended to convey changes in the entity’s schema,
	// invariants, or external contract. Typical representations include:
	//
	//   - Simple labels: "v1", "v2".
	//   - Semantic versions: "v1.2.0".
	//   - Date-based versions: "2024-01-15".
	//
	// This value can be used by:
	//
	//   - Migration tools and schema registries.
	//   - Backwards-compatibility checks.
	//   - Client libraries that need to adapt to different entity versions.
	//
	// Recommended properties:
	//
	//   - MUST change when the schema or externally visible contract of
	//     the entity changes in an incompatible way.
	//   - SHOULD remain constant across deployments of the same build.
	//   - SHOULD be machine-readable enough to allow simple equality or
	//     ordering checks, where applicable.
	//
	// # Contract
	//
	//   - The returned string MAY be empty if versioning is not relevant
	//     or not modeled, but callers SHOULD treat the empty string as
	//     "version unknown" rather than "no version".
	//   - The implementation MUST be safe for concurrent use and MUST NOT
	//     perform blocking I/O or heavyweight computations.
	//   - Implementations SHOULD prefer returning a constant or precomputed
	//     version string tied to the build or schema definition, rather than
	//     deriving it at runtime.
	EntityVersion() string
}
