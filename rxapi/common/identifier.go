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

// Identifier extends Namer with a per-instance identifier.
//
// # Overview
//
// Identifier is an extended naming contract that combines:
//
//   - A type-level, canonical entity name (via Namer.EntityName), and
//   - An instance-level identifier (via EntityID).
//
// This is particularly useful for logging, tracing, auditing, and debugging,
// where it is often necessary to distinguish not only *what kind* of entity
// is involved, but also *which specific instance* participated in an event.
//
// The type-level name and the instance-level identifier are conceptually
// orthogonal:
//
//   - EntityName describes the logical kind, class, or category of the
//     entity (for example, "domain.user").
//   - EntityID distinguishes one instance of that kind from another
//     (for example, "user-123" or "123").
//
// # Usage
//
// A typical pattern is to implement both Namer and Identifier on the same
// domain type:
//
//   type User struct {
//       ID   string
//       Name string
//   }
//
//   func (u User) EntityName() string { return "domain.user" }
//   func (u User) EntityID() string   { return u.ID }
//
//   user := User{ID: "123", Name: "Alice"}
//   // user.EntityName() -> "domain.user"
//   // user.EntityID()   -> "123"
//
// Callers MAY use EntityName for high-level grouping (e.g., log fields like
// "entity" or "resource") and EntityID for correlation across requests,
// traces, or log entries (e.g., "entity_id", "user_id", "order_id").
type Identifier interface {
	Namer

	// EntityID returns a stable identifier for this entity instance.
	//
	// # Semantics
	//
	// EntityID is an instance-level counterpart to EntityName:
	//
	//   - EntityName identifies the *type* or *kind* of entity.
	//   - EntityID identifies a particular instance of that type.
	//
	// The returned value is intended to be:
	//
	//   - Stable for the lifetime of the instance (MUST).
	//   - Unique within the scope of the corresponding EntityName (SHOULD),
	//     or at least unique within the domain where it is used for
	//     correlation (for example, per tenant or per environment).
	//   - Safe to expose in logs and traces, subject to application-specific
	//     privacy and security constraints (MUST be considered by the
	//     implementation).
	//
	// Implementations MAY return an empty string to indicate that the
	// instance does not have a meaningful identifier (for example, ephemeral
	// or anonymous entities). Callers MUST be prepared to handle the empty
	// string as "no ID" and SHOULD NOT assume non-emptiness unless explicitly
	// guaranteed by the domain model.
	//
	// # Contract
	//
	//   - EntityID MUST be deterministic for a given instance over its
	//     lifetime (no spontaneous changes).
	//   - EntityID MUST be safe for concurrent calls from multiple goroutines.
	//   - EntityID SHOULD avoid heap allocations on the hot path (for example,
	//     by returning a field or a precomputed value).
	//   - EntityID MUST NOT perform blocking operations or I/O.
	//   - EntityID MUST be reasonably cheap to compute; if the identifier is
	//     derived from expensive state, it SHOULD be precomputed and cached.
	//
	// # Usage in infrastructure
	//
	// Logging, tracing, and metrics layers MAY use the combination of
	// (EntityName, EntityID) as a composite key for:
	//
	//   - Correlating events that involve the same instance.
	//   - Building structured fields such as "entity" and "entity_id".
	//   - Tagging spans, metrics, or audit records with stable identifiers.
	//
	// However, infrastructure MUST NOT rely on EntityID being globally
	// unique across all EntityName values unless such a property is
	// explicitly documented by the application or implementation.
	EntityID() string
}
