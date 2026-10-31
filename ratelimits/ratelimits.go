// Package ratelimits declares the Birdeye request budgets this plugin paces
// itself against. The number is the Standard (free, $0) plan tier — the
// published default; users on Lite/Starter/Premium/Business raise it via
// Terminal's Rate Limits UI override.
//
// Sources (retrieved 2026-10-31):
//   - https://data.birdeye.so/docs/guides/api-access/rate-limiting/index.md
//     ("Per User Account Rate Limit": Standard 1 rps, Lite/Starter 15 rps,
//     Premium 50 rps, Business 100 rps — applied per account, i.e. per API
//     key, across all APIs)
//   - https://data.birdeye.so/docs/guides/payment/pricing.md
//     (Standard = $0 tier)
//   - https://data.birdeye.so/docs/guides/data-accessibility-by-packages.md
//     (every endpoint this plugin calls — /defi/ohlcv, /defi/ohlcv/base_quote,
//     /defi/v2/markets, /defi/v3/search — is Standard-accessible)
package ratelimits

import (
	"time"

	"github.com/plusev-terminal/go-plugin-common/meta"
)

// BirdeyeAPI returns the manifest rate-limit rules for public-api.birdeye.so.
// Birdeye enforces its account limit per API key, so the rule is
// PerCredential.
func BirdeyeAPI() []meta.RateLimitRule {
	return []meta.RateLimitRule{
		// Standard (free) plan: 1 request/second per account (API key),
		// applied across all endpoints.
		// https://data.birdeye.so/docs/guides/api-access/rate-limiting/index.md
		meta.NewRateLimitRule("account", 1, time.Second).
			WithLabel("Standard (free) plan, per API key").
			PerCredential(),
	}
}
