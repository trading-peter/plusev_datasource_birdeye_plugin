package main

import (
	"sort"
	"strings"
	"time"

	ex "github.com/plusev-terminal/go-plugin-common/datasrc/exchange"
	"github.com/plusev-terminal/go-plugin-common/logging"
	m "github.com/plusev-terminal/go-plugin-common/meta"
	"github.com/plusev-terminal/go-plugin-common/plugin"
	"github.com/plusev-terminal/go-plugin-common/requester"
	"github.com/plusev-terminal/go-plugin-common/trading"

	"github.com/trading-peter/plusev_datasource_birdeye_plugin/birdeye"
	"github.com/trading-peter/plusev_datasource_birdeye_plugin/ratelimits"
)

// ============================================================================
// Main - Register the plugin
// ============================================================================

func init() {
	p := &BirdeyePlugin{}
	plugin.RegisterPlugin(p)
}

func main() {}

// ============================================================================
// Plugin Implementation
// ============================================================================

var log = logging.NewLogger("birdeye")

type BirdeyePlugin struct {
	config *plugin.ConfigStore
	client *birdeye.Client
}

func (p *BirdeyePlugin) GetMeta() m.Meta {
	return m.Meta{
		PluginID:    "birdeye-datasource",
		Name:        "Birdeye DEX Prices",
		AppID:       "datasrc",
		Category:    "dex",
		Description: "Historical OHLCV price data via Birdeye. Supports tokens across Solana, Ethereum, Base, Arbitrum, and more. Ideal for DEX token pricing and cost basis calculation.",
		Author:      "trading_peter",
		Version:     "v1.0.0",
		Repository:  "https://github.com/trading-peter/plusev_datasource_birdeye_plugin",
		Tags:        []string{"birdeye", "dex", "ohlcv", "solana", "ethereum", "defi"},
		Contacts: []m.AuthorContact{
			{Kind: "x.com", Value: "https://x.com/trading_peter"},
		},
		Resources: m.ResourceAccess{
			AllowedNetworkTargets: []m.NetworkTargetRule{
				{
					Pattern: "https://public-api.birdeye.so/*",
					// Free-tier budgets with sources; see package ratelimits.
					Limits: ratelimits.BirdeyeAPI(),
				},
			},
		},
		Features: []string{},
	}
}

func (p *BirdeyePlugin) GetConfigFields() []plugin.ConfigField {
	return birdeye.GetConfigFields()
}

func (p *BirdeyePlugin) OnInit(config *plugin.ConfigStore) error {
	p.config = config

	apiKey := config.GetString("apiKey")
	chain := config.GetString("chain")
	if chain == "" {
		chain = "solana"
	}

	p.client = birdeye.NewClient(requester.NewRequester(), apiKey, chain)

	return nil
}

func (p *BirdeyePlugin) OnShutdown() error {
	return nil
}

func (p *BirdeyePlugin) GetRateLimits() []plugin.RateLimit {
	// Opt out of the legacy per-command limiter: public-api.birdeye.so is paced
	// by the manifest network-target limits declared in GetMeta, and an empty
	// (but non-nil) return keeps the export while avoiding double throttling.
	return []plugin.RateLimit{}
}

func (p *BirdeyePlugin) RegisterCommands(router *plugin.CommandRouter) {
	router.Register(ex.CMD_GET_MARKET, p.handleGetMarket)
	router.Register(ex.CMD_GET_TIMEFRAMES, p.handleGetTimeframes)
	router.Register(ex.CMD_GET_OHLCV, p.handleGetOHLCV)
}

// ============================================================================
// Command Handlers
// ============================================================================

func (p *BirdeyePlugin) handleGetMarket(params map[string]any) plugin.Response {
	mp := ex.GetMarketParamsFromMap(params)
	if err := mp.Validate(); err != nil {
		return plugin.ErrorResponseMsg(err.Error())
	}

	var tokenAddress string
	var tokenSymbol string

	if mp.Address != "" {
		tokenAddress = mp.Address
	} else if mp.Symbol != "" {
		// Extract the base symbol from a pair (e.g. "SOL/USDC" -> "SOL").
		base := mp.Symbol
		if idx := strings.Index(base, "/"); idx > 0 {
			base = base[:idx]
		}

		token, err := p.client.SearchToken(base)
		if err != nil {
			return plugin.ErrorResponse(err)
		}

		tokenAddress = token.Address
		tokenSymbol = token.Symbol
	} else {
		return plugin.ErrorResponseMsg("symbol or address is required")
	}

	// Fetch real DEX markets for this token.
	markets, err := p.client.GetMarketsForToken(tokenAddress)
	if err != nil {
		return plugin.ErrorResponse(err)
	}

	// Pick the first market (already sorted by liquidity desc).
	m := markets[0]

	// If the user requested a specific quote (e.g. "SOL/USDC"), try to find a matching market.
	if mp.Symbol != "" {
		parts := strings.SplitN(mp.Symbol, "/", 2)

		if len(parts) == 2 && parts[1] != "" {
			wantQuote := strings.ToUpper(parts[1])

			for _, candidate := range markets {
				if strings.ToUpper(candidate.Quote.Symbol) == wantQuote {
					m = candidate
					break
				}
			}
		}
	}

	baseSymbol := m.Base.Symbol
	quoteSymbol := m.Quote.Symbol

	// The API may omit symbol for some tokens. Fall back to the search-resolved symbol
	// or extract from the market name (e.g. "JitoSOL-SOL").
	if baseSymbol == "" {
		if tokenSymbol != "" && strings.EqualFold(m.Base.Address, tokenAddress) {
			baseSymbol = tokenSymbol
		} else if m.Name != "" {
			parts := strings.SplitN(m.Name, "-", 2)
			if len(parts) == 2 {
				baseSymbol = parts[0]
			}
		}
	}

	if quoteSymbol == "" {
		if tokenSymbol != "" && strings.EqualFold(m.Quote.Address, tokenAddress) {
			quoteSymbol = tokenSymbol
		} else if m.Name != "" {
			parts := strings.SplitN(m.Name, "-", 2)
			if len(parts) == 2 {
				quoteSymbol = parts[1]
			}
		}
	}

	pairLabel := baseSymbol + "/" + quoteSymbol

	return plugin.SuccessResponse(trading.Market{
		Symbol:    pairLabel,
		Label:     pairLabel,
		Address:   m.Base.Address + "|" + m.Quote.Address,
		Base:      baseSymbol,
		Quote:     quoteSymbol,
		AssetType: "spot",
	}, time.Hour)
}

func (p *BirdeyePlugin) handleGetTimeframes(params map[string]any) plugin.Response {
	return plugin.SuccessResponse(p.client.GetTimeframes(), time.Hour*24)
}

func (p *BirdeyePlugin) handleGetOHLCV(params map[string]any) plugin.Response {
	ohlcvParams := ex.GetOHLCVParamsFromMap(params)

	if ohlcvParams.Market.Address == "" {
		return plugin.ErrorResponseMsg("market.address is required - use a market returned by getMarket")
	}

	if ohlcvParams.Timeframe == "" {
		return plugin.ErrorResponseMsg("timeframe is required")
	}

	var startUnix, endUnix int64

	if ohlcvParams.StartTime != nil {
		startUnix = ohlcvParams.StartTime.Unix()
	}

	if ohlcvParams.EndTime != nil {
		endUnix = ohlcvParams.EndTime.Unix()
	} else {
		endUnix = time.Now().Unix()
	}

	var records []trading.OHLCVRecord
	var err error

	// Check if the address contains a pipe separator (base|quote).
	if strings.Contains(ohlcvParams.Market.Address, "|") {
		addrParts := strings.SplitN(ohlcvParams.Market.Address, "|", 2)
		records, err = p.client.GetOHLCVBaseQuote(addrParts[0], addrParts[1], ohlcvParams.Timeframe, startUnix, endUnix, ohlcvParams.Limit)
	} else {
		records, err = p.client.GetOHLCV(ohlcvParams.Market.Address, ohlcvParams.Timeframe, startUnix, endUnix, ohlcvParams.Limit)
	}

	if err != nil {
		log.ErrorWithData("handleGetOHLCV error", map[string]any{
			"error":     err.Error(),
			"address":   ohlcvParams.Market.Address,
			"timeframe": ohlcvParams.Timeframe,
		})

		return plugin.ErrorResponse(err)
	}

	// Sort ascending by default.
	sort.Slice(records, func(i, j int) bool {
		return records[i].OpenTime < records[j].OpenTime
	})

	if ohlcvParams.Sort == "desc" {
		for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
			records[i], records[j] = records[j], records[i]
		}
	}

	if ohlcvParams.Limit > 0 && len(records) > ohlcvParams.Limit {
		records = records[:ohlcvParams.Limit]
	}

	if ohlcvParams.CacheForSeconds > 0 {
		cacheDuration := time.Duration(ohlcvParams.CacheForSeconds) * time.Second
		return plugin.SuccessResponse(records, cacheDuration)
	}

	return plugin.SuccessResponse(records)
}
