package flux

import (
	"encoding/json"

	"github.com/adityaxdiwakar/tda-go"
	"github.com/gorilla/websocket"
)

// Session is the session object for the flux driver and can be created and
// returned using flux.New()
type Session struct {
	TdaSession              tda.Session
	wsConn                  *websocket.Conn
	ConfigUrl               string
	CurrentState            storedCache
	TransactionChannel      chan storedCache
	ChartRequestVers        map[string]int
	SearchRequestVers       map[string]int
	OptionSeriesRequestVers map[string]int
	MutexLock               bool
	HandlerWorking          bool
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
