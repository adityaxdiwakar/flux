package flux

type storedCache struct {
	Chart          ChartStoredCache          `json:"chart"`
	Search         searchStoredCache         `json:"search"`
	Quote          QuoteStoredCache          `json:"quote"`
	OptionSeries   optionSeriesValue         `json:"optionSeries"`
	OptionChainGet optionChainGetStoredCache `json:"optionChainGet"`
}
