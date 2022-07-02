package flux

import (
	"encoding/json"
	"hash/fnv"
	"sync"

	"github.com/adityaxdiwakar/tda-go"
	"github.com/gorilla/websocket"
)

// Session is the session object for the flux driver and can be created and
// returned using flux.New()
type Session struct {
	TdaSession                tda.Session
	wsConn                    *websocket.Conn
	ConfigURL                 string
	Cache                     cache
	NotificationChannel       chan bool
	ChartRequestVers          map[string]int
	QuoteRequestVers          map[string]int
	SearchRequestVers         map[string]int
	OptionSeriesRequestVers   map[string]int
	OptionChainGetRequestVers map[string]int
	OptionQuoteRequestVers    map[string]int
	ChartRouteTable           ChartRouteTableT
	Mu                        sync.Mutex
	QuoteMu                   sync.Mutex
	MutexLock                 bool
	HandlerWorking            bool
	DebugFlag                 bool
	Established               bool
}

func (s *Session) specHash(str string) int {
	h := fnv.New32a()
	h.Write([]byte(str))
	return int(h.Sum32()) - (1<<31 - 1)
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
