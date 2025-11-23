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

package strategy_test

import (
	"testing"

	"dirpx.dev/rfx/rxapi/cache/strategy"
)

// TestStrategyString verifies that String() returns the expected stable
// tokens for all known strategy.Strategy values and a diagnostic form for unknown
// values.
func TestStrategyString(t *testing.T) {
	tests := []struct {
		name     string
		strategy strategy.Strategy
		want     string
	}{
		{
			name:     "LRU",
			strategy: strategy.LRU,
			want:     "LRU",
		},
		{
			name:     "LFU",
			strategy: strategy.LFU,
			want:     "LFU",
		},
		{
			name:     "TTL",
			strategy: strategy.TTL,
			want:     "TTL",
		},
		{
			name:     "None",
			strategy: strategy.None,
			want:     "None",
		},
		{
			name:     "Unknown",
			strategy: strategy.Strategy(42),
			want:     "Unknown(42)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.strategy.String()
			if got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParseStrategyValid verifies that strategy.Parse correctly parses all
// supported textual representations in a case-insensitive way and with
// optional surrounding whitespace.
func TestParseStrategyValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  strategy.Strategy
	}{
		{"LRU upper", "LRU", strategy.LRU},
		{"LRU lower", "lru", strategy.LRU},
		{"LRU mixed", "lRu", strategy.LRU},
		{"LRU trimmed", "  lru  ", strategy.LRU},

		{"LFU upper", "LFU", strategy.LFU},
		{"LFU lower", "lfu", strategy.LFU},

		{"TTL upper", "TTL", strategy.TTL},
		{"TTL lower", "ttl", strategy.TTL},

		{"None canonical", "None", strategy.None},
		{"None lower", "none", strategy.None},
		{"None trimmed", "  none  ", strategy.None},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := strategy.Parse(tt.input)
			if err != nil {
				t.Fatalf("strategy.Parse(%q) error = %v, want nil", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("strategy.Parse(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseStrategyInvalid verifies that strategy.Parse rejects invalid input,
// returns a non-nil error, and does not rely on the returned strategy.Strategy value
// in the error case.
func TestParseStrategyInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Empty", ""},
		{"Whitespace", "   "},
		{"Unknown token", "invalid"},
		{"Partial match", "LRU1"},
		{"Garbage", "!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := strategy.Parse(tt.input)
			if err == nil {
				t.Fatalf("strategy.Parse(%q) error = nil, want non-nil", tt.input)
			}
			// The contract says callers MUST NOT rely on got in error case, but
			// current implementation returns strategy.None. We can assert this
			// to keep tests in sync with implementation, while still treating
			// it as an implementation detail.
			if got != strategy.None {
				t.Fatalf("strategy.Parse(%q) = %v, want strategy.None on error", tt.input, got)
			}
		})
	}
}

// TestMustParseStrategyValid verifies that strategy.MustParse behaves like
// strategy.Parse on valid inputs and does not panic.
func TestMustParseStrategyValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  strategy.Strategy
	}{
		{"LRU", "LRU", strategy.LRU},
		{"LFU", "lfu", strategy.LFU},
		{"TTL", "ttl", strategy.TTL},
		{"None", "None", strategy.None},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.MustParse(tt.input)
			if got != tt.want {
				t.Fatalf("strategy.MustParse(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestMustParseStrategyInvalid verifies that strategy.MustParse panics on
// invalid input, as documented.
func TestMustParseStrategyInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Empty", ""},
		{"Invalid token", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("strategy.MustParse(%q) did not panic on invalid input", tt.input)
				}
			}()
			_ = strategy.MustParse(tt.input)
		})
	}
}

// TestStrategyMarshalTextValid verifies that MarshalText returns the canonical
// string tokens for all known strategies, with no error.
func TestStrategyMarshalTextValid(t *testing.T) {
	tests := []struct {
		name     string
		strategy strategy.Strategy
		want     string
	}{
		{"LRU", strategy.LRU, "LRU"},
		{"LFU", strategy.LFU, "LFU"},
		{"TTL", strategy.TTL, "TTL"},
		{"None", strategy.None, "None"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBytes, err := tt.strategy.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText(%v) error = %v, want nil", tt.strategy, err)
			}
			got := string(gotBytes)
			if got != tt.want {
				t.Fatalf("MarshalText(%v) = %q, want %q", tt.strategy, got, tt.want)
			}
		})
	}
}

// TestStrategyMarshalTextUnknown verifies that MarshalText fails for unknown
// strategy.Strategy values and does not silently serialize them.
func TestStrategyMarshalTextUnknown(t *testing.T) {
	var s strategy.Strategy = strategy.Strategy(42)

	got, err := s.MarshalText()
	if err == nil {
		t.Fatalf("MarshalText(%v) error = nil, want non-nil for unknown strategy", s)
	}
	if got != nil && len(got) != 0 {
		t.Fatalf("MarshalText(%v) = %q, want nil/empty on error", s, string(got))
	}
}

// TestStrategyUnmarshalTextValid verifies that UnmarshalText accepts all
// supported tokens (case-insensitive) and sets the receiver accordingly.
func TestStrategyUnmarshalTextValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  strategy.Strategy
	}{
		{"LRU", "LRU", strategy.LRU},
		{"lru lowercase", "lru", strategy.LRU},
		{"LFU", "LFU", strategy.LFU},
		{"TTL", "ttl", strategy.TTL},
		{"None canonical", "None", strategy.None},
		{"none lowercase", "none", strategy.None},
		{"trimmed", "  ttl  ", strategy.TTL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s strategy.Strategy

			if err := s.UnmarshalText([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalText(%q) error = %v, want nil", tt.input, err)
			}
			if s != tt.want {
				t.Fatalf("UnmarshalText(%q) = %v, want %v", tt.input, s, tt.want)
			}
		})
	}
}

// TestStrategyUnmarshalTextInvalid verifies that UnmarshalText rejects invalid
// input, returns an error, and does not modify the receiver.
func TestStrategyUnmarshalTextInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Empty", ""},
		{"Whitespace", "   "},
		{"Unknown token", "invalid"},
		{"Garbage", "!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start from a known value to verify that it is not changed on error.
			var s strategy.Strategy = strategy.LRU

			err := s.UnmarshalText([]byte(tt.input))
			if err == nil {
				t.Fatalf("UnmarshalText(%q) error = nil, want non-nil", tt.input)
			}
			if s != strategy.LRU {
				t.Fatalf("UnmarshalText(%q) modified receiver to %v, want %v on error", tt.input, s, strategy.LRU)
			}
		})
	}
}

// TestStrategyMarshalUnmarshalRoundTrip verifies that a strategy.Strategy value can be
// marshaled and then unmarshaled back to the same value for all known
// strategies.
func TestStrategyMarshalUnmarshalRoundTrip(t *testing.T) {
	strategies := []strategy.Strategy{
		strategy.LRU,
		strategy.LFU,
		strategy.TTL,
		strategy.None,
	}

	for _, original := range strategies {
		t.Run(original.String(), func(t *testing.T) {
			data, err := original.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText(%v) error = %v, want nil", original, err)
			}

			var decoded strategy.Strategy
			if err := decoded.UnmarshalText(data); err != nil {
				t.Fatalf("UnmarshalText(%q) error = %v, want nil", string(data), err)
			}

			if decoded != original {
				t.Fatalf("round-trip: got %v, want %v", decoded, original)
			}
		})
	}
}
