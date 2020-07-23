package flux

import (
	"encoding/json"
	"fmt"

	"github.com/adityaxdiwakar/tda-go"
)

type Session struct {
	TdaSession tda.Session
}

type ChartRequestSignature struct {
	Ticker string
	Range  string
	Width  string
}

type ChartSocketRequest struct {
	Service           string   `json:"service"`
	ID                string   `json:"id"`
	Ver               int      `json:"ver"`
	Symbol            string   `json:"symbol"`
	AggregationPeriod string   `json:"aggregationPeriod"`
	Studies           []string `json:"studies"`
	Range             string   `json:"range"`
}

type ChartSocketRequests struct {
	Payload []ChartSocketRequest `json:"payload"`
}

func (c *ChartRequestSignature) GenerateShortName() string {
	return fmt.Sprintf("%s@%s:%s", c.Ticker, c.Range, c.Width)
}

func (c *ChartRequestSignature) GenerateSocketRequest() (*string, error) {
	req := ChartSocketRequests{
		Payload: []ChartSocketRequest{
			{
				Service:           "chart",
				ID:                "chart",
				Ver:               0,
				Symbol:            c.Ticker,
				AggregationPeriod: c.Width,
				Studies:           []string{},
				Range:             c.Range,
			},
		},
	}
	reqBytes, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	reqStr := string(reqBytes)
	return &reqStr, nil
}
