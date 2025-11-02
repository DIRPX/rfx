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

package config_test

import (
	"testing"

	"dirpx.dev/rfx/config"
)

func TestDefaultConfigValues(t *testing.T) {
	got := config.DefaultConfig()

	if got.IncludeBuiltins != config.DefaultIncludeBuiltins {
		t.Fatalf("IncludeBuiltins = %v, want %v", got.IncludeBuiltins, config.DefaultIncludeBuiltins)
	}
	if got.MaxUnwrap != config.DefaultMaxUnwrap {
		t.Fatalf("MaxUnwrap = %d, want %d", got.MaxUnwrap, config.DefaultMaxUnwrap)
	}
	if got.MapPreferElem != config.DefaultMapPreferElem {
		t.Fatalf("MapPreferElem = %v, want %v", got.MapPreferElem, config.DefaultMapPreferElem)
	}
}

func TestNewConfig_NoOptions_EqualsDefault(t *testing.T) {
	def := config.DefaultConfig()
	got := config.NewConfig()
	if got != def {
		t.Fatalf("NewConfig() = %+v, want default %+v", got, def)
	}
}

func TestWithIncludeBuiltins(t *testing.T) {
	c := config.NewConfig(config.WithIncludeBuiltins(false))
	if c.IncludeBuiltins {
		t.Fatalf("IncludeBuiltins = %v, want false", c.IncludeBuiltins)
	}

	c2 := config.NewConfig(config.WithIncludeBuiltins(true))
	if !c2.IncludeBuiltins {
		t.Fatalf("IncludeBuiltins = %v, want true", c2.IncludeBuiltins)
	}
}

func TestWithMapPreferElem(t *testing.T) {
	c := config.NewConfig(config.WithMapPreferElem(false))
	if c.MapPreferElem {
		t.Fatalf("MapPreferElem = %v, want false", c.MapPreferElem)
	}

	c2 := config.NewConfig(config.WithMapPreferElem(true))
	if !c2.MapPreferElem {
		t.Fatalf("MapPreferElem = %v, want true", c2.MapPreferElem)
	}
}

func TestWithMaxUnwrap_Positive(t *testing.T) {
	c := config.NewConfig(config.WithMaxUnwrap(3))
	if c.MaxUnwrap != 3 {
		t.Fatalf("MaxUnwrap = %d, want 3", c.MaxUnwrap)
	}
}

func TestWithMaxUnwrap_Negative_ResetsToDefault(t *testing.T) {
	c := config.NewConfig(config.WithMaxUnwrap(-1))
	if c.MaxUnwrap != config.DefaultMaxUnwrap {
		t.Fatalf("MaxUnwrap = %d, want default %d", c.MaxUnwrap, config.DefaultMaxUnwrap)
	}
}

func TestOptionsOrder_LastWins(t *testing.T) {
	c := config.NewConfig(
		config.WithIncludeBuiltins(false),
		config.WithIncludeBuiltins(true),
		config.WithMaxUnwrap(2),
		config.WithMaxUnwrap(5),
		config.WithMapPreferElem(false),
		config.WithMapPreferElem(true),
	)

	if !c.IncludeBuiltins {
		t.Errorf("IncludeBuiltins = %v, want true (last option wins)", c.IncludeBuiltins)
	}
	if c.MaxUnwrap != 5 {
		t.Errorf("MaxUnwrap = %d, want 5 (last option wins)", c.MaxUnwrap)
	}
	if !c.MapPreferElem {
		t.Errorf("MapPreferElem = %v, want true (last option wins)", c.MapPreferElem)
	}
}

func TestNewConfig_Guardrails_MaxUnwrapZeroAllowed(t *testing.T) {
	// The constructor only resets negative values. Zero is allowed by design.
	c := config.NewConfig(config.WithMaxUnwrap(0))
	if c.MaxUnwrap != 0 {
		t.Fatalf("MaxUnwrap = %d, want 0 (zero is allowed)", c.MaxUnwrap)
	}
}
