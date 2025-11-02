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

package config

import (
	"dirpx.dev/rfx/apis"
)

const (
	// DefaultIncludeBuiltins represents the default for IncludeBuiltins.
	// When true, built-in types will be included.
	DefaultIncludeBuiltins = true
	// DefaultMaxUnwrap represents the default for MaxUnwrap.
	// A value of 8 should be sufficient for all practical purposes.
	DefaultMaxUnwrap = 8
	// DefaultMapPreferElem represents the default for MapPreferElem.
	// When true, map value types are preferred when searching for named inner types.
	DefaultMapPreferElem = true
)

// NewConfig constructs an apis.Config from the given options.
func NewConfig(opts ...Option) apis.Config {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	// Ensure MaxUnwrap is valid.
	if cfg.MaxUnwrap < 0 {
		cfg.MaxUnwrap = DefaultMaxUnwrap
	}
	return cfg
}

// DefaultConfig is the default configuration used when none is provided.
func DefaultConfig() apis.Config {
	return apis.Config{
		IncludeBuiltins: DefaultIncludeBuiltins,
		MaxUnwrap:       DefaultMaxUnwrap,
		MapPreferElem:   DefaultMapPreferElem,
	}
}

// Option is a functional option that mutates an apis.Config during construction.
type Option func(*apis.Config)

// WithIncludeBuiltins sets the IncludeBuiltins option.
func WithIncludeBuiltins(include bool) Option {
	return func(c *apis.Config) {
		c.IncludeBuiltins = include
	}
}

// WithMaxUnwrap sets the MaxUnwrap option.
// A negative value resets to the default.
func WithMaxUnwrap(max int) Option {
	return func(c *apis.Config) {
		if max < 0 {
			c.MaxUnwrap = DefaultMaxUnwrap
			return
		}
		c.MaxUnwrap = max
	}
}

// WithMapPreferElem sets the MapPreferElem option.
func WithMapPreferElem(prefer bool) Option {
	return func(c *apis.Config) {
		c.MapPreferElem = prefer
	}
}
