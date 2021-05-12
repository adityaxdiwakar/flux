package flux

type storedCache struct {
	Chart          ChartStoredCache          `json:"chart"`
	Search         searchStoredCache         `json:"search"`
	OptionSeries   optionSeriesValue         `json:"optionSeries"`
	OptionChainGet optionChainGetStoredCache `json:"optionChainGet"`
}
