package birdeye

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/plusev-terminal/go-plugin-common/v2/logging"
	"github.com/plusev-terminal/go-plugin-common/v2/plugin"
	rt "github.com/plusev-terminal/go-plugin-common/v2/requester/types"
	"github.com/plusev-terminal/go-plugin-common/v2/trading"
)

const baseURL = "https://public-api.birdeye.so"

var log = logging.NewLogger("birdeye")

// Client handles all HTTP communication with the Birdeye API.
type Client struct {
	requester rt.RequestDoer
	apiKey    string
	chain     string
}

// NewClient creates a new Birdeye API client.
func NewClient(req rt.RequestDoer, apiKey, chain string) *Client {
	return &Client{
		requester: req,
		apiKey:    apiKey,
		chain:     chain,
	}
}

// headers returns the standard request headers including authentication.
func (c *Client) headers() map[string]string {
	return map[string]string{
		"Accept":    "application/json",
		"X-API-KEY": c.apiKey,
		"x-chain":   c.chain,
	}
}

// GetConfigFields returns the configuration fields for this plugin.
func GetConfigFields() []plugin.ConfigField {
	return []plugin.ConfigField{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        "text",
			Required:    true,
			Encrypt:     true,
			Mask:        true,
			Description: "Your Birdeye API key. Get one at https://birdeye.so",
		},
		{
			Name:          "chain",
			Label:         "Blockchain Network",
			Type:          "select",
			Required:      true,
			Description:   "Blockchain network to query token data on.",
			DefaultOption: "solana",
			Options: []plugin.ConfigFieldSelectOption{
				{Value: "solana", Label: "Solana"},
				{Value: "ethereum", Label: "Ethereum"},
				{Value: "base", Label: "Base"},
				{Value: "arbitrum", Label: "Arbitrum"},
				{Value: "bsc", Label: "BNB Smart Chain"},
				{Value: "polygon", Label: "Polygon"},
				{Value: "avalanche", Label: "Avalanche"},
				{Value: "optimism", Label: "Optimism"},
				{Value: "zksync", Label: "zkSync"},
				{Value: "sui", Label: "Sui"},
				{Value: "aptos", Label: "Aptos"},
			},
		},
	}
}

// ============================================================================
// Token Discovery
// ============================================================================

// GetMarketsForToken fetches DEX markets/pools for a given token address
// using /defi/v2/markets, sorted by liquidity descending.
// Returns the list of markets found.
func (c *Client) GetMarketsForToken(address string) ([]MarketItem, error) {
	url := fmt.Sprintf("%s/defi/v2/markets?address=%s&sort_by=liquidity&sort_type=desc&limit=20", baseURL, address)

	var resp MarketsResponse
	_, err := c.requester.Send(&rt.Request{
		URL:     url,
		Method:  "GET",
		Headers: c.headers(),
	}, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch markets: %w", err)
	}

	if !resp.Success || len(resp.Data.Items) == 0 {
		return nil, fmt.Errorf("no markets found for token %s", address)
	}

	return resp.Data.Items, nil
}

// SearchToken searches for a token by keyword and returns the best match.
func (c *Client) SearchToken(keyword string) (SearchToken, error) {
	url := fmt.Sprintf("%s/defi/v3/search?chain=%s&keyword=%s&target=token&sort_by=volume_24h_usd&sort_type=desc&limit=10&search_by=symbol",
		baseURL, c.chain, keyword)

	var resp SearchResponse
	_, err := c.requester.Send(&rt.Request{
		URL:     url,
		Method:  "GET",
		Headers: c.headers(),
	}, &resp)
	if err != nil {
		return SearchToken{}, fmt.Errorf("token search failed: %w", err)
	}

	upperKeyword := strings.ToUpper(keyword)

	// Prefer exact symbol match.
	for _, item := range resp.Data.Items {
		if item.Type != "token" {
			continue
		}

		for _, token := range item.Result {
			if strings.ToUpper(token.Symbol) == upperKeyword {
				return token, nil
			}
		}
	}

	// Fall back to the first token result if no exact match.
	for _, item := range resp.Data.Items {
		if item.Type != "token" || len(item.Result) == 0 {
			continue
		}

		return item.Result[0], nil
	}

	return SearchToken{}, fmt.Errorf("no token found for %q on %s", keyword, c.chain)
}

// ============================================================================
// OHLCV Data
// ============================================================================

// supportedTimeframes lists all timeframe values accepted by the Birdeye API.
var supportedTimeframes = map[string]bool{
	"1m": true, "3m": true, "5m": true, "15m": true, "30m": true,
	"1H": true, "2H": true, "4H": true, "6H": true, "8H": true, "12H": true,
	"1D": true, "3D": true, "1W": true, "1M": true,
}

// GetOHLCV fetches OHLCV data for a single token (USD-denominated).
// address is the token contract address.
// timeframe is one of the supported Birdeye timeframes (e.g. "1m", "1H", "1D").
func (c *Client) GetOHLCV(address, timeframe string, startUnix, endUnix int64, limit int) ([]trading.OHLCVRecord, error) {
	if !supportedTimeframes[timeframe] {
		return nil, fmt.Errorf("unsupported timeframe: %s", timeframe)
	}

	var allRecords []trading.OHLCVRecord
	maxPerRequest := 1000
	maxPages := 10
	currentStart := startUnix

	for page := 0; page < maxPages; page++ {
		url := fmt.Sprintf("%s/defi/ohlcv?address=%s&type=%s&time_from=%d&time_to=%d",
			baseURL, address, timeframe, currentStart, endUnix)

		var resp OHLCVResponse
		_, err := c.requester.Send(&rt.Request{
			URL:     url,
			Method:  "GET",
			Headers: c.headers(),
		}, &resp)
		if err != nil {
			return nil, fmt.Errorf("OHLCV request failed: %w", err)
		}

		if !resp.Success || len(resp.Data.Items) == 0 {
			break
		}

		for _, item := range resp.Data.Items {
			allRecords = append(allRecords, trading.OHLCVRecord{
				OpenTime: item.UnixTime,
				Open:     floatToStr(item.Open),
				High:     floatToStr(item.High),
				Low:      floatToStr(item.Low),
				Close:    floatToStr(item.Close),
				Volume:   floatToStr(item.Volume),
				IsClosed: true,
			})
		}

		if limit > 0 && len(allRecords) >= limit {
			allRecords = allRecords[:limit]
			break
		}

		if len(resp.Data.Items) < maxPerRequest {
			break
		}

		// Advance the start cursor past the newest candle returned.
		newestTime := resp.Data.Items[len(resp.Data.Items)-1].UnixTime
		if newestTime <= currentStart {
			break
		}

		currentStart = newestTime + 1

		if currentStart >= endUnix {
			break
		}
	}

	return allRecords, nil
}

// GetOHLCVBaseQuote fetches OHLCV data for a base/quote token pair.
func (c *Client) GetOHLCVBaseQuote(baseAddress, quoteAddress, timeframe string, startUnix, endUnix int64, limit int) ([]trading.OHLCVRecord, error) {
	if !supportedTimeframes[timeframe] {
		return nil, fmt.Errorf("unsupported timeframe: %s", timeframe)
	}

	var allRecords []trading.OHLCVRecord
	maxPerRequest := 1000
	maxPages := 10
	currentStart := startUnix

	for page := 0; page < maxPages; page++ {
		url := fmt.Sprintf("%s/defi/ohlcv/base_quote?base_address=%s&quote_address=%s&type=%s&time_from=%d&time_to=%d",
			baseURL, baseAddress, quoteAddress, timeframe, currentStart, endUnix)

		var resp OHLCVBaseQuoteResponse
		_, err := c.requester.Send(&rt.Request{
			URL:     url,
			Method:  "GET",
			Headers: c.headers(),
		}, &resp)
		if err != nil {
			return nil, fmt.Errorf("OHLCV base/quote request failed: %w", err)
		}

		if !resp.Success || len(resp.Data.Items) == 0 {
			break
		}

		for _, item := range resp.Data.Items {
			allRecords = append(allRecords, trading.OHLCVRecord{
				OpenTime: item.UnixTime,
				Open:     floatToStr(item.Open),
				High:     floatToStr(item.High),
				Low:      floatToStr(item.Low),
				Close:    floatToStr(item.Close),
				Volume:   floatToStr(item.VolumeBase),
				IsClosed: true,
			})
		}

		if limit > 0 && len(allRecords) >= limit {
			allRecords = allRecords[:limit]
			break
		}

		if len(resp.Data.Items) < maxPerRequest {
			break
		}

		newestTime := resp.Data.Items[len(resp.Data.Items)-1].UnixTime
		if newestTime <= currentStart {
			break
		}

		currentStart = newestTime + 1

		if currentStart >= endUnix {
			break
		}
	}

	return allRecords, nil
}

// GetTimeframes returns the list of supported OHLCV timeframes.
func (c *Client) GetTimeframes() []string {
	return []string{"1m", "3m", "5m", "15m", "30m", "1H", "2H", "4H", "6H", "8H", "12H", "1D", "3D", "1W", "1M"}
}

// ============================================================================
// Helpers
// ============================================================================

// floatToStr converts a float64 to a string, preserving full precision.
func floatToStr(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
