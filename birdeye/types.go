package birdeye

// ============================================================================
// Birdeye API response types
// https://docs.birdeye.so/reference
// ============================================================================

// MarketsResponse is the response from /defi/v2/markets.
type MarketsResponse struct {
	Success bool        `json:"success"`
	Data    MarketsData `json:"data"`
}

// MarketsData wraps the list of market items.
type MarketsData struct {
	Items []MarketItem `json:"items"`
	Total int          `json:"total"`
}

// MarketItem represents a single DEX market/pool returned by /defi/v2/markets.
type MarketItem struct {
	Address   string          `json:"address"`
	Name      string          `json:"name"`
	Source    string          `json:"source"`
	Base      MarketTokenInfo `json:"base"`
	Quote     MarketTokenInfo `json:"quote"`
	Liquidity float64         `json:"liquidity"`
	Volume24h float64         `json:"volume24h"`
}

// MarketTokenInfo holds base or quote token info within a market item.
// Note: Symbol and Icon may be empty for some tokens in the API response.
type MarketTokenInfo struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	Icon     string `json:"icon"`
}

// SearchResponse is the response from /defi/v3/search.
type SearchResponse struct {
	Success bool       `json:"success"`
	Data    SearchData `json:"data"`
}

// SearchData wraps the search result items.
type SearchData struct {
	Items []SearchItem `json:"items"`
}

// SearchItem groups results by type (e.g. "token", "market").
type SearchItem struct {
	Type   string        `json:"type"`
	Result []SearchToken `json:"result"`
}

// SearchToken represents a single token in search results.
type SearchToken struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	Network  string `json:"network"`
	Decimals int    `json:"decimals"`
}

// OHLCVResponse is the response from /defi/ohlcv (single token, USD-denominated).
type OHLCVResponse struct {
	Success bool      `json:"success"`
	Data    OHLCVData `json:"data"`
}

// OHLCVData wraps the OHLCV items.
type OHLCVData struct {
	Items []OHLCVItem `json:"items"`
}

// OHLCVItem is a single candlestick for the single-token OHLCV endpoint.
type OHLCVItem struct {
	Open     float64 `json:"o"`
	High     float64 `json:"h"`
	Low      float64 `json:"l"`
	Close    float64 `json:"c"`
	Volume   float64 `json:"v"`
	UnixTime int64   `json:"unixTime"`
}

// OHLCVBaseQuoteResponse is the response from /defi/ohlcv/base_quote.
type OHLCVBaseQuoteResponse struct {
	Success bool               `json:"success"`
	Data    OHLCVBaseQuoteData `json:"data"`
}

// OHLCVBaseQuoteData wraps the base/quote OHLCV items.
type OHLCVBaseQuoteData struct {
	Items []OHLCVBaseQuoteItem `json:"items"`
}

// OHLCVBaseQuoteItem is a single candlestick for the base/quote OHLCV endpoint.
type OHLCVBaseQuoteItem struct {
	Open        float64 `json:"o"`
	High        float64 `json:"h"`
	Low         float64 `json:"l"`
	Close       float64 `json:"c"`
	VolumeBase  float64 `json:"vBase"`
	VolumeQuote float64 `json:"vQuote"`
	UnixTime    int64   `json:"unixTime"`
}
