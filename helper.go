package flux

import (
	"encoding/json"
	"fmt"
)

func (p *protocolResponse) IsEmpty() bool {
	if p.Build == "" {
		return true
	} else if p.Session == "" {
		return true
	} else if p.Ver == "" {
		return true
	} else {
		return false
	}
}

func (s *Session) dataAsChartObject() (*CachedData, error) {
	data := KeyCachedData{}
	err := json.Unmarshal(s.CurrentState, &data)
	if err != nil {
		return nil, err
	}
	return &data.Data, nil
}

// Gateway returns the gateway URL as a string for the live trading connection
// to be made
func (s *Session) Gateway() (string, error) {
	res, err := s.TdaSession.HttpClient.Get("https://trade.thinkorswim.com/v1/api/config")
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", ErrGatewayUnsuccessful
	}

	defer res.Body.Close()

	var gatewayResponse gatewayConfigResponse

	json.NewDecoder(res.Body).Decode(&gatewayResponse)
	return gatewayResponse.MobileGatewayURL.Livetrading, nil
}

func (c *ChartRequestSignature) shortName() string {
	return fmt.Sprintf("%s@%s:%s", c.Ticker, c.Range, c.Width)
}
