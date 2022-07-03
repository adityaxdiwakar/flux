package flux

import (
	"encoding/json"
	"hash/fnv"
	"sync"

	"github.com/adityaxdiwakar/tda-go"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/gorilla/websocket"
)

// Session is the session object for the flux driver and can be created and
// returned using flux.New()
type Session struct {
	wsConn     *websocket.Conn
	TdaSession tda.Session
	Lock       sync.Mutex
	ConfigURL  string
	Cache      cache

	// Session flags
	Restarting  bool
	Reconnecter bool
	DebugFlag   bool
	Established bool

	// Route tables
	ChartRouteTable  RouteTable[ChartData]
	SearchRouteTable RouteTable[SearchData]

	// Request version numbers
	ChartRequestVers  map[string]int
	SearchRequestVers map[string]int
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

func applyPatch[T any](body Body[T], state *cache) error {
	paylodBytes, err := json.Marshal(body.Patches)
	if err != nil {
		return err
	}

	jspatch, err := jsonpatch.DecodePatch(paylodBytes)
	if err != nil {
		return err
	}

	byteState, err := json.Marshal(*state)
	if err != nil {
		return err
	}

	byteState, err = jspatch.Apply(byteState)
	if err != nil {
		return err
	}

	json.Unmarshal(byteState, state)
	return nil
}
