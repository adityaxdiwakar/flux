package flux

type storedCache struct {
	Chart  chartStoredCache  `json:"chart"`
	Search searchStoredCache `json:"search"`
}
