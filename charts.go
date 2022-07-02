package flux

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
)

// ChartRequestSignature is the parameter for a chart request
type ChartRequestSignature struct {
	// ticker for the content of a chart request
	Ticker string

	// range is the timeframe for chart data to be received, see specs.txt
	Range string

	// width is the width of the candles to be received, see specs.txt
	Width string

	// internal use only
	UniqueID string
}

// shortname presented as CHART#TICKER@RANGE:WIDTH
func (c *ChartRequestSignature) shortName() string {
	return fmt.Sprintf("CHART#%s@%s:%s", c.Ticker, c.Range, c.Width)
}

// ChartStoredCache is an object containing what is returned from a chart request
type ChartData struct {
	Symbol  string `json:"symbol"`
	Candles struct {
		Timestamps []int64   `json:"timestamps"`
		Opens      []float64 `json:"opens"`
		Highs      []float64 `json:"highs"`
		Lows       []float64 `json:"lows"`
		Closes     []float64 `json:"closes"`
		Volumes    []float64 `json:"volumes"`
	} `json:"candles"`
	RequestID  string `json:"requestId"`
	RequestVer int    `json:"requestVer"`
}

func (s *Session) chartHandler(data Payload) {
	route, ok := s.ChartRouteTable[data.Header.ID]
	if !ok {
		// Receiving a response for an ID that isn't in the route table
		// probably means the request context timed out before receiving
		// this, so we can ignore this.
		return
	}
	resp := RespTuple[ChartData]{}

	if data.Header.Type == "error" {
		body := ErroredBody{}
		if err := json.Unmarshal(data.Body, &body); err != nil {
			resp.Err = errors.New("Received an errored response that could not be parsed")
		} else {
			resp.Err = errors.New(body.Message)
		}
		route <- resp
		return
	}

	if data.Header.Type == "snapshot" {
		body := ChartData{}
		if err := json.Unmarshal(data.Body, &body); err != nil {
			resp.Err = errors.New("Received a snapshot response that could not be parsed")
			route <- resp
			return
		}
		body.RequestID = data.Header.ID
		body.RequestVer = data.Header.Ver
		s.Cache.Chart = body

		resp.Body = s.Cache.Chart
		resp.Err = nil
		route <- resp

		return
	}

	if data.Header.Type != "patch" {
		resp.Err = errors.New("Response body cannot be understood: neither error nor patch")
		route <- resp
		return
	}

	body := Body[ChartData]{}
	if err := json.Unmarshal(data.Body, &body); err != nil {
		resp.Err = errors.New("Response body could not be interpreted as type Body")
		route <- resp
		return
	}

	newState := s.Cache
	for idx := 0; idx < len(body.Patches); idx++ {
		// TODO: handle the "error" path case (might be deprecated)

		if body.Patches[idx].Path == "" {
			newState.Chart = ChartData{}
		}
		body.Patches[idx].Path = "/chart" + body.Patches[idx].Path
	}

	applyPatch(body, &newState)

	s.Cache = newState
	s.Cache.Chart.RequestID = data.Header.ID
	s.Cache.Chart.RequestVer = data.Header.Ver

	resp.Body = s.Cache.Chart
	resp.Err = nil
	route <- resp
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

// RequestChart takes a ChartRequestSignature as an input and responds with a
// ChartStoredCache object, it utilizes the cached if it can (with updated diffs), or
// else it makes a new request and waits for it - if a ticker does not load in
// time, ErrNotReceviedInTime is sent as an error
func (s *Session) RequestChart(specs ChartRequestSignature) (*ChartData, error) {

	// force capitalization of tickers, since the socket is case sensitive
	specs.Ticker = strings.ToUpper(specs.Ticker)

	if s.Cache.Chart.RequestID == specs.shortName() {
		return &s.Cache.Chart, nil
	}

	uniqueID := fmt.Sprintf("%s-%d", specs.shortName(),
		s.ChartRequestVers[specs.shortName()])

	payload := gatewayRequestLoad{
		Payload: []gatewayRequest{
			{
				Header: gatewayHeader{
					Service: "chart",
					ID:      uniqueID,
					Ver:     int(s.ChartRequestVers[specs.shortName()]),
				},
				Params: gatewayParams{
					Symbol:            specs.Ticker,
					AggregationPeriod: specs.Width,
					Range:             specs.Range,
					Studies:           []string{},
					ExtendedHours:     true,
				},
			},
		},
	}

	s.ChartRequestVers[specs.shortName()]++
	s.ChartRouteTable[uniqueID] = make(chan RespTuple[ChartData])
	s.sendJSON(payload)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	select {

	case recvPayload := <-s.ChartRouteTable[uniqueID]:
		return &recvPayload.Body, recvPayload.Err

	case <-ctx.Done():
		return nil, ErrNotReceivedInTime
	}

}
