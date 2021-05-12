package flux

type storedCache struct {
	Chart  ChartStoredCache  `json:"chart"`
	Search searchStoredCache `json:"search"`
}
