package rfx

import (
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	apis "dirpx.dev/rfx/apis"
)

// ---------------------- Helpers ----------------------

func itoa(i int) string { return fmtInt(i) }

func fmtInt(i int) string {
	if i == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	n := i
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
func boolToChar(b bool) string {
	if b {
		return "T"
	}
	return "F"
}
func intToChar(i int) string { return fmtInt(i) }

// Reset to a clean snapshot using our test builder.
// This fully replaces builder, config, ext and rebuilds registry/resolver.
// Pins are reset (preg=false, pres=false) because we pass nil reg/res.
func resetWithBuilder(tb testing.TB, b apis.Builder, cfg apis.Config, ext any) {
	tb.Helper()
	SetAll(&cfg, ext, nil, nil, b)
}

// ---------------------- Test doubles (mocks) ----------------------

type mockRegistry struct {
	id   string
	mu   sync.Mutex
	data map[reflect.Type]string
}

func newMockRegistry(id string) *mockRegistry {
	return &mockRegistry{id: id, data: make(map[reflect.Type]string)}
}

func (m *mockRegistry) Register(t reflect.Type, name string) error {
	m.mu.Lock()
	m.data[t] = name
	m.mu.Unlock()
	return nil
}
func (m *mockRegistry) Lookup(t reflect.Type) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, ok := m.data[t]
	return n, ok
}
func (m *mockRegistry) Entries() []apis.Entry {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []apis.Entry
	for t, n := range m.data {
		out = append(out, apis.Entry{Type: t, Name: n})
	}
	return out
}
func (m *mockRegistry) Count() int { m.mu.Lock(); defer m.mu.Unlock(); return len(m.data) }
func (m *mockRegistry) Reset()     { m.mu.Lock(); m.data = make(map[reflect.Type]string); m.mu.Unlock() }

type mockResolver struct {
	id       string
	resolveC int
	mu       sync.Mutex
}

func (r *mockResolver) Resolve(v any, cfg apis.Config) string {
	r.mu.Lock()
	r.resolveC++
	r.mu.Unlock()
	return r.id + ":" + boolToChar(cfg.IncludeBuiltins) + ":" + boolToChar(cfg.MapPreferElem) + ":" + intToChar(cfg.MaxUnwrap)
}

func (r *mockResolver) ResolveType(t reflect.Type, cfg apis.Config) string {
	return r.Resolve(nil, cfg) + ":" + t.String()
}

type mockBuilder struct {
	mu             sync.Mutex
	lastCfg        apis.Config
	lastExt        any
	lastPrevRegID  string
	lastPrevResID  string
	regCounter     int
	resCounter     int
	returnFixedReg apis.Registry // optional override
	returnFixedRes apis.Resolver // optional override
}

func (b *mockBuilder) BuildRegistry(cfg apis.Config, prev apis.Registry, ext any) apis.Registry {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastCfg, b.lastExt = cfg, ext
	if prev != nil {
		if mr, ok := prev.(*mockRegistry); ok {
			b.lastPrevRegID = mr.id
		}
	}
	if b.returnFixedReg != nil {
		return b.returnFixedReg
	}
	b.regCounter++
	return newMockRegistry("reg#" + itoa(b.regCounter))
}

func (b *mockBuilder) BuildResolver(cfg apis.Config, reg apis.Registry, prev apis.Resolver, ext any) apis.Resolver {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastCfg, b.lastExt = cfg, ext
	if prev != nil {
		if mr, ok := prev.(*mockResolver); ok {
			b.lastPrevResID = mr.id
		}
	}
	if b.returnFixedRes != nil {
		return b.returnFixedRes
	}
	b.resCounter++
	return &mockResolver{id: "res#" + itoa(b.resCounter)}
}

// ---------------------- Tests ----------------------

func TestSetConfig_Rebuilds_Unpinned(t *testing.T) {
	b := &mockBuilder{}
	resetWithBuilder(t, b, apis.Config{IncludeBuiltins: false, MapPreferElem: true, MaxUnwrap: 8}, nil)

	// snapshot 1
	s1Reg := Registry() // was DefaultRegistry()
	s1Res := Resolver() // was DefaultResolver()

	// change cfg -> both should rebuild (not pinned)
	SetConfig(apis.Config{IncludeBuiltins: true, MapPreferElem: false, MaxUnwrap: 4})

	s2Reg := Registry()
	s2Res := Resolver()

	if s1Reg == s2Reg {
		t.Fatalf("registry was not rebuilt on SetConfig (unpinned)")
	}
	if s1Res == s2Res {
		t.Fatalf("resolver was not rebuilt on SetConfig (unpinned)")
	}

	b.mu.Lock()
	gotCfg := b.lastCfg
	b.mu.Unlock()
	if gotCfg.MaxUnwrap != 4 || !gotCfg.IncludeBuiltins || gotCfg.MapPreferElem {
		t.Fatalf("builder received wrong cfg: %+v", gotCfg)
	}
}

func TestSetRegistry_PinsRegistry_and_RebuildsResolverIfUnpinned(t *testing.T) {
	b := &mockBuilder{}
	resetWithBuilder(t, b, apis.Config{IncludeBuiltins: false, MapPreferElem: true, MaxUnwrap: 8}, nil)

	customReg := newMockRegistry("custom")
	SetRegistry(customReg)

	beforeRes := Resolver()
	SetConfig(apis.Config{IncludeBuiltins: true, MapPreferElem: true, MaxUnwrap: 8})

	afterReg := Registry()
	afterRes := Resolver()

	if afterReg != customReg {
		t.Fatalf("pinned registry was rebuilt unexpectedly")
	}
	if afterRes == beforeRes {
		t.Fatalf("resolver was not rebuilt when cfg changed and res not pinned")
	}
}

func TestSetResolver_PinsResolver(t *testing.T) {
	b := &mockBuilder{}
	resetWithBuilder(t, b, apis.Config{IncludeBuiltins: false, MapPreferElem: true, MaxUnwrap: 8}, nil)

	// Pin resolver
	customRes := &mockResolver{id: "custom"}
	SetResolver(customRes)

	// Grab current registry pointer (should be from builder b)
	regBefore := Registry()

	// Change cfg -> expect: registry rebuilt (not pinned), resolver unchanged (pinned)
	SetConfig(apis.Config{IncludeBuiltins: true, MapPreferElem: true, MaxUnwrap: 8})

	regAfter := Registry()
	resAfter := Resolver()

	if resAfter != customRes {
		t.Fatalf("pinned resolver was rebuilt unexpectedly")
	}
	if regAfter == regBefore {
		t.Fatalf("registry was not rebuilt on SetConfig when resolver is pinned")
	}
}

func TestSetBuilder_Rebuilds_Only_Unpinned(t *testing.T) {
	// Start with builder A
	a := &mockBuilder{}
	resetWithBuilder(t, a, apis.Config{IncludeBuiltins: false, MapPreferElem: true, MaxUnwrap: 8}, nil)

	// Pin resolver, leave registry unpinned
	SetResolver(&mockResolver{id: "pinned"})
	regBefore := Registry()
	resBefore := Resolver()

	// Swap to builder B (no rebuild yet per current semantics)
	b := &mockBuilder{}
	SetBuilder(b)

	// Trigger rebuild by changing config -> expect: registry rebuilt (unpinned), resolver unchanged (pinned)
	SetConfig(apis.Config{IncludeBuiltins: true, MapPreferElem: false, MaxUnwrap: 6})

	regAfter := Registry()
	resAfter := Resolver()

	if regAfter == regBefore {
		t.Fatalf("registry did not rebuild after SetBuilder + SetConfig (unpinned)")
	}
	if resAfter != resBefore {
		t.Fatalf("pinned resolver was rebuilt after SetBuilder + SetConfig")
	}
}

func TestSetExt_Rebuilds_Unpinned_and_PassesValue(t *testing.T) {
	// Ensure snapshot uses our mock builder
	b := &mockBuilder{}
	resetWithBuilder(t, b, apis.Config{IncludeBuiltins: false, MapPreferElem: true, MaxUnwrap: 8}, nil)

	// Change ext -> should rebuild unpinned layers via current builder (b) and pass ext
	type extCfg struct{ X int }
	SetExt(extCfg{X: 42})

	b.mu.Lock()
	got := b.lastExt
	b.mu.Unlock()
	ec, ok := got.(extCfg)
	if !ok || ec.X != 42 {
		t.Fatalf("builder did not receive ext properly: %#v", got)
	}

	// Pin both and ensure no rebuild on SetExt
	SetRegistry(Registry())
	SetResolver(Resolver())
	rCntBefore, sCntBefore := func() (int, int) {
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.regCounter, b.resCounter
	}()
	SetExt(extCfg{X: 7})
	rCntAfter, sCntAfter := func() (int, int) {
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.regCounter, b.resCounter
	}()
	if rCntAfter != rCntBefore || sCntAfter != sCntBefore {
		t.Fatalf("SetExt should not rebuild when both layers are pinned")
	}
}

func TestUnpin_Allows_Rebuild_After(t *testing.T) {
	b := &mockBuilder{}
	resetWithBuilder(t, b, apis.Config{IncludeBuiltins: false, MapPreferElem: true, MaxUnwrap: 8}, nil)

	SetRegistry(Registry())
	SetResolver(Resolver())

	reg1 := Registry()
	res1 := Resolver()
	SetConfig(apis.Config{IncludeBuiltins: true, MapPreferElem: false, MaxUnwrap: 4})
	if Registry() != reg1 || Resolver() != res1 {
		t.Fatalf("pinned layers should not rebuild on SetConfig")
	}

	UnpinRegistry()
	UnpinResolver()
	SetConfig(apis.Config{IncludeBuiltins: false, MapPreferElem: false, MaxUnwrap: 6})
	if Registry() == reg1 {
		t.Fatalf("registry should rebuild after UnpinRegistry+SetConfig")
	}
	if Resolver() == res1 {
		t.Fatalf("resolver should rebuild after UnpinResolver+SetConfig")
	}
}

func TestEntity_Concurrent_With_SetConfig(t *testing.T) {
	b := &mockBuilder{}
	resetWithBuilder(t, b, apis.Config{IncludeBuiltins: false, MapPreferElem: true, MaxUnwrap: 8}, nil)

	type token struct{}
	done := make(chan struct{})
	var wg sync.WaitGroup

	readers := runtime.GOMAXPROCS(0) * 4
	wg.Add(readers)
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				_ = Entity(token{})
				_ = EntityType(reflect.TypeOf(token{}))
			}
		}()
	}

	go func() {
		for i := 0; i < 20; i++ {
			SetConfig(apis.Config{
				IncludeBuiltins: i%2 == 0,
				MapPreferElem:   i%3 == 0,
				MaxUnwrap:       4 + (i % 5),
			})
			time.Sleep(time.Millisecond)
		}
		close(done)
	}()

	wg.Wait()
	<-done
}
