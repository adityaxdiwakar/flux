package flux

type storedCache struct {
	Chart          ChartStoredCache          `json:"chart"`
	Search         SearchStoredCache         `json:"search"`
	Quote          QuoteStoredCache          `json:"quote"`
	OptionSeries   OptionSeriesCache         `json:"optionSeries"`
	OptionChainGet OptionChainGetStoredCache `json:"optionChainGet"`
}
