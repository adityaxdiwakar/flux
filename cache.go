package flux

type storedCache struct {
	Chart          ChartData                 `json:"chart"`
	Search         SearchStoredCache         `json:"search"`
	Quote          QuoteStoredCache          `json:"quote"`
	OptionSeries   OptionSeriesCache         `json:"optionSeries"`
	OptionChainGet OptionChainGetStoredCache `json:"optionChainGet"`
	OptionQuote    OptionQuoteCache          `json:"optionQuote"`
}
