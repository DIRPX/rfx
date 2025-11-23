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

package strategy

import (
	"fmt"
	"strings"
)

// Strategy controls the eviction and expiration policy of a cache.
//
// # Overview
//
// Strategy is a small enumerated type that describes how a cache instance
// manages its entries over time. It governs which items are retained,
// which items are evicted, and under what conditions. Concrete cache
// implementations use this value to select the underlying algorithm and
// associated data structures.
//
// Strategy is intentionally minimal and format-agnostic: it does not define
// specific capacity limits, TTL durations, or implementation details, but
// instead selects a broad class of behavior (e.g., LRU vs LFU vs TTL).
//
// # Values
//
// The following strategies are defined:
//
//   - LRU  — Least Recently Used eviction.
//   - LFU  — Least Frequently Used eviction.
//   - TTL  — Time-To-Live based expiration.
//   - None — Caching disabled (pass-through behavior).
//
// Implementations MAY support additional, implementation-specific tuning
// parameters (such as capacity, TTL duration, or sampling strategy), but
// those are configured separately from Strategy.
//
// # Contract
//
//   - Cache implementations MUST treat Strategy as a stable, public API;
//     adding new values is allowed, but existing values MUST NOT change
//     their semantics in breaking ways.
//   - Strategy values MUST be safe to use concurrently across goroutines
//     (they are plain integers).
//   - Strategy SHOULD be used as an input to configuration or factory code,
//     not mutated at runtime in performance-critical paths.
type Strategy int

const (
	// LRU selects Least Recently Used eviction policy.
	//
	// # Semantics
	//
	// Under LRU, when the cache needs to evict entries (for example, because
	// it has reached a capacity limit), it SHOULD evict the entry that has
	// not been accessed for the longest time. "Access" typically includes:
	//
	//   - Reads (cache hits).
	//   - Writes/insertions (puts/sets).
	//
	// Recommended usage:
	//
	//   - General-purpose request/response caching.
	//   - Workloads where "recently used" is a good predictor of "soon to be
	//     used again".
	//
	// Implementation notes:
	//
	//   - Commonly implemented via a combination of a hashmap and a linked
	//     list or a similar structure that tracks recency.
	//   - Implementations SHOULD maintain LRU metadata in a way that is safe
	//     for concurrent access if the cache itself is concurrent.
	LRU Strategy = iota

	// LFU selects Least Frequently Used eviction policy.
	//
	// # Semantics
	//
	// Under LFU, when the cache needs to evict entries, it SHOULD evict the
	// entry with the lowest observed access frequency. Implementations MAY
	// use approximate algorithms (for example, sampling or bucketed counters)
	// to balance accuracy against overhead.
	//
	// Recommended usage:
	//
	//   - Workloads with "hot" keys that are accessed significantly more
	//     often than others, where preserving frequently-used entries is
	//     more important than preserving recently-used ones.
	//
	// Implementation notes:
	//
	//   - Requires tracking access counts or approximate frequencies.
	//   - Implementations SHOULD document whether frequency counts are
	//     resettable or decayed over time to avoid unbounded growth.
	LFU

	// TTL selects a Time-To-Live based expiration policy.
	//
	// # Semantics
	//
	// Under TTL, entries are associated with an absolute or relative expiry
	// time. Once an entry has exceeded its configured TTL, it MUST be treated
	// as expired:
	//
	//   - Lookups SHOULD NOT return expired entries.
	//   - Maintenance routines MAY remove expired entries eagerly, lazily,
	//     or on-demand during lookups.
	//
	// TTL does not, by itself, define:
	//
	//   - The actual TTL duration.
	//   - Whether capacity-based eviction is also applied when the cache is
	//     full and all entries are "fresh".
	//
	// These aspects MUST be configured separately by the cache implementation
	// or its caller.
	//
	// Recommended usage:
	//
	//   - Caching data that becomes stale after a fixed time window (for
	//     example, configuration snapshots, discovery information, or
	//     external API results).
	TTL

	// None disables caching for the associated cache instance.
	//
	// # Semantics
	//
	// When None is selected, the cache MUST NOT retain entries across
	// calls in a way that affects observable behavior. In practice, this
	// usually means:
	//
	//   - Reads always result in a miss from the perspective of the caller.
	//   - Writes either no-op or are immediately "evicted" so that they are
	//     not visible to future reads.
	//
	// None is primarily useful for:
	//
	//   - Testing or debugging, to compare behavior with and without caching.
	//   - Environments where caching is undesirable (for example, very small
	//     deployments or extremely dynamic data).
	//
	// Implementations MAY still maintain internal statistics or metrics
	// (e.g., count "would-be" cache accesses) as long as this does not
	// introduce observable caching semantics for callers.
	None
)

// String returns a human-readable representation of the Strategy value.
//
// # Semantics
//
// String implements fmt.Stringer and provides short, stable identifiers
// suitable for logging, metrics labels, configuration dumps, and debugging.
// For all defined enum values, the returned strings are:
//
//   - LRU  -> "LRU"
//   - LFU  -> "LFU"
//   - TTL  -> "TTL"
//   - None -> "None"
//
// For unknown or out-of-range values, String returns a diagnostic form
// "Unknown(<n>)", where <n> is the underlying integer value. This behavior
// is intentional and MUST NOT panic, so that corrupted or unexpected values
// can still be surfaced safely in logs and diagnostics.
//
// # Contract
//
//   - The mapping from known Strategy values to strings MUST remain stable;
//     changing the spelling or casing is a breaking change for systems that
//     persist or parse these strings.
//   - Callers MAY use the returned string for display or logging, but they
//     SHOULD NOT rely on it as a primary configuration format unless this
//     is explicitly documented and properly versioned.
func (cs Strategy) String() string {
	switch cs {
	case LRU:
		return "LRU"
	case LFU:
		return "LFU"
	case TTL:
		return "TTL"
	case None:
		return "None"
	default:
		return fmt.Sprintf("Unknown(%d)", cs)
	}
}

// Parse parses a textual representation of a Strategy.
//
// # Overview
//
// Parse converts a string token into the corresponding Strategy
// value. It accepts the same canonical tokens that are produced by
// Strategy.String() for known values, with case-insensitive matching.
//
// Accepted (case-insensitive) inputs:
//
//   - "LRU"  -> LRU
//   - "LFU"  -> LFU
//   - "TTL"  -> TTL
//   - "None" -> None
//
// Any other input results in a non-nil error.
//
// # Contract
//
//   - s MAY contain surrounding whitespace; it will be trimmed.
//   - On success, Parse returns a valid Strategy and a nil error.
//   - On failure, Parse returns None and a non-nil error;
//     callers MUST NOT rely on the returned Strategy value in the error case.
//   - Parse MUST NOT panic for any input.
//
// # Usage
//
// Parse is suitable for parsing configuration values, environment
// variables, CLI flags, and other human-authored or external inputs. For
// hard-coded values that are expected to be valid, callers MAY prefer
// MustParse for brevity.
//
// Example:
//
//	strategy, err := Parse("lru")
//	if err != nil {
//	    // handle invalid configuration
//	}
//
//	_ = strategy // LRU
func Parse(s string) (Strategy, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return None, fmt.Errorf("cache: empty strategy")
	}

	switch strings.ToUpper(trimmed) {
	case "LRU":
		return LRU, nil
	case "LFU":
		return LFU, nil
	case "TTL":
		return TTL, nil
	case "NONE":
		return None, nil
	default:
		return None, fmt.Errorf("cache: unknown strategy %q", s)
	}
}

// MustParse is like Parse but panics on invalid input.
//
// # Overview
//
// MustParse is a convenience helper for contexts where the input
// string is expected to be a valid Strategy token and encountering an invalid
// value is considered a programmer error rather than a recoverable condition.
//
// It is intended for:
//
//   - Hard-coded configuration in Go code.
//   - Tests and examples.
//   - Initialization code where failing fast with a panic is acceptable.
//
// # Contract
//
//   - On valid input, MustParse returns the same value as
//     Parse and MUST NOT panic.
//   - On invalid input (including empty strings), MustParse panics
//     with a diagnostic message.
//   - Callers MUST NOT use MustParse on untrusted or user-supplied
//     data; they SHOULD use Parse instead and handle errors.
//
// # Usage
//
//	var defaultStrategy = MustParse("LRU")
func MustParse(s string) Strategy {
	strategy, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return strategy
}

// MarshalText encodes Strategy as text.
//
// # Overview
//
// MarshalText implements encoding.TextMarshaler for Strategy. It converts
// a Strategy value into its canonical textual representation, suitable for
// use in textual encodings such as:
//
//   - encoding/json (when using ",string" struct tags or custom handling),
//   - encoding/xml,
//   - encoding/yaml (via third-party libraries),
//   - configuration files and human-readable dumps.
//
// For all defined Strategy values, MarshalText returns the same tokens as
// Strategy.String() for known values ("LRU", "LFU", "TTL", "None").
//
// # Contract
//
//   - On success, MarshalText returns a non-nil byte slice and a nil error.
//   - For unknown or out-of-range Strategy values, MarshalText returns a
//     non-nil error and MUST NOT silently serialize an "Unknown(...)" form;
//     this avoids persisting potentially invalid states.
//   - MarshalText MUST NOT panic for any Strategy value.
//
// # Usage
//
// MarshalText is typically called indirectly by encoding frameworks. Direct
// callers MAY use it when they need an explicit textual form:
//
//	b, err := strategy.MarshalText()
//	if err != nil {
//	    // handle unknown/invalid strategy
//	}
//	fmt.Println(string(b)) // e.g. "LRU"
func (cs Strategy) MarshalText() ([]byte, error) {
	switch cs {
	case LRU, LFU, TTL, None:
		return []byte(cs.String()), nil
	default:
		return nil, fmt.Errorf("cache: cannot marshal unknown strategy %d", cs)
	}
}

// UnmarshalText decodes a Strategy from its textual representation.
//
// # Overview
//
// UnmarshalText implements encoding.TextUnmarshaler for Strategy. It accepts
// the same textual tokens as Parse, with case-insensitive matching:
//
//   - "LRU"  -> LRU
//   - "LFU"  -> LFU
//   - "TTL"  -> TTL
//   - "None" -> None
//
// Leading and trailing whitespace are ignored. Any other value results in
// a non-nil error, and the target is left unchanged.
//
// # Contract
//
//   - text MAY contain surrounding whitespace; it will be trimmed.
//   - On success, *cs is set to the parsed value and a nil error is returned.
//   - On failure, *cs MUST NOT be modified and a non-nil error is returned.
//   - UnmarshalText MUST NOT panic for any input.
//   - Callers MUST NOT assume that an empty text slice is valid; it is
//     treated as an error.
//
// # Usage
//
// UnmarshalText is typically invoked by encoding frameworks when decoding
// configuration or serialized state. It can also be used directly:
//
//	var strategy Strategy
//	if err := strategy.UnmarshalText([]byte("ttl")); err != nil {
//	    // handle invalid input
//	}
//
//	_ = strategy // TTL
func (cs *Strategy) UnmarshalText(text []byte) error {
	trimmed := strings.TrimSpace(string(text))
	if trimmed == "" {
		return fmt.Errorf("cache: empty strategy")
	}

	value, err := Parse(trimmed)
	if err != nil {
		return err
	}

	*cs = value
	return nil
}
