package ratelimits

import (
	"testing"
	"time"

	"github.com/plusev-terminal/go-plugin-common/v2/meta"
)

// Pins the declared Birdeye budget so it cannot silently drift.
// Source (2026-10-31):
// https://data.birdeye.so/docs/guides/api-access/rate-limiting/index.md
// Standard (free) plan: 1 request/second per account, i.e. per API key.
func TestBirdeyeAPIRules(t *testing.T) {
	rules := BirdeyeAPI()
	want := []meta.RateLimitRule{
		{
			ID:       "account",
			Label:    "Standard (free) plan, per API key",
			Limit:    1,
			WindowMs: time.Second.Milliseconds(),
			Scope:    meta.RateLimitScopeCredential,
		},
	}
	if len(rules) != len(want) {
		t.Fatalf("got %d rules, want %d: %+v", len(rules), len(want), rules)
	}
	for i, r := range rules {
		w := want[i]
		if r.ID != w.ID || r.Label != w.Label || r.Limit != w.Limit ||
			r.WindowMs != w.WindowMs || r.Scope != w.Scope || r.PerKeyEnabled != w.PerKeyEnabled {
			t.Errorf("rule %d = %+v, want %+v", i, r, w)
		}
		if r.Burst != 0 {
			t.Errorf("rule %d Burst = %d, want 0 (defaults to Limit)", i, r.Burst)
		}
		if len(r.Paths) != 0 {
			t.Errorf("rule %d Paths = %v, want none (applies to the whole host)", i, r.Paths)
		}
		if r.CostWeighted {
			t.Errorf("rule %d must not be weighted", i)
		}
	}
}
