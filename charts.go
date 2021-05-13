package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
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

type ChartStoredCache struct {
	Symbol     string    `json:"symbol"`
	Timestamps []int64   `json:"timestamps"`
	Open       []float64 `json:"open"`
	High       []float64 `json:"high"`
	Low        []float64 `json:"low"`
	Close      []float64 `json:"close"`
	Volume     []int     `json:"volume"`
	Events     []struct {
		Symbol      string `json:"symbol"`
		CompanyName string `json:"companyName"`
		Website     string `json:"website"`
		IsActual    bool   `json:"isActual"`
		Time        int64  `json:"time"`
	} `json:"events"`
	Service    string `json:"service"`
	RequestID  string `json:"requestId"`
	RequestVer int    `json:"requestVer"`
}

type newChartObject struct {
	Op    string           `json:"op"`
	Path  string           `json:"path"`
	Value ChartStoredCache `json:"value"`
}

type updateChartObject struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value int    `json:"value"`
}

func (s *Session) chartHandler(msg []byte, gab *gabs.Container) {

	// get the request id from the header
	rID := gab.Search("header", "id").String()
	rID = rID[1 : len(rID)-1]
	rService := gab.Search("header", "service").String()
	rService = rService[1 : len(rService)-1]

	// get the patches
	patches := gab.S("body", "patches").Children()
	for _, patch := range patches {
		// TODO: implement actual error handling, currently using log.Fatal()
		// which is bad
		var err error
		bytesJson := patch.Bytes()

		var modifiedChart []byte

		if patch.S("path").String() == "/error" {
			continue
		}

		if patch.S("path").String() == `""` {
			newChart := newChartObject{}
			json.Unmarshal(bytesJson, &newChart)
			newChart.Path = "/chart"
			newChart.Value.RequestID = rID
			newChart.Value.Service = rService
			modifiedChart, err = json.Marshal([]newChartObject{newChart})
		} else {
			if rID != fmt.Sprintf("\"%s\"", s.CurrentState.Chart.RequestID) {
				continue
			}
			updatedChart := updateChartObject{}
			json.Unmarshal(bytesJson, &updatedChart)
			updatedChart.Path = "/chart" + updatedChart.Path
			modifiedChart, err = json.Marshal([]updateChartObject{updatedChart})
		}

		if err != nil {
			return
		}
		jspatch, err := jsonpatch.DecodePatch(modifiedChart)
		if err != nil {
			return
		}

		byteState, _ := json.Marshal(s.CurrentState)

		byteState, err = jspatch.Apply(byteState)
		if err != nil {
			return
		}

		var newState storedCache
		json.Unmarshal(byteState, &newState)

		s.CurrentState = newState

		if patch.S("path").String() == `""` {
			s.TransactionChannel <- newState
		}
	}
}

// RequestChart takes a ChartRequestSignature as an input and responds with a
// ChartStoredCache object, it utilizes the cached if it can (with updated diffs), or
// else it makes a new request and waits for it - if a ticker does not load in
// time, ErrNotReceviedInTime is sent as an error
func (s *Session) RequestChart(specs ChartRequestSignature) (*ChartStoredCache, error) {

	// force capitalization of tickers, since the socket is case sensitive
	specs.Ticker = strings.ToUpper(specs.Ticker)

	if s.CurrentState.Chart.RequestID == specs.shortName() {
		return &s.CurrentState.Chart, nil
	}

	uniqueID := fmt.Sprintf("%s-%d", specs.shortName(), s.ChartRequestVers[specs.shortName()])

	payload := gatewayRequestLoad{
		Payload: []gatewayRequest{
			{
				Header: gatewayHeader{
					Service: "chart",
					ID:      uniqueID,
					Ver:     s.ChartRequestVers[specs.shortName()],
				},
				Params: gatewayParams{
					Symbol:            specs.Ticker,
					AggregationPeriod: specs.Width,
					Studies:           []string{},
					Range:             specs.Range,
				},
			},
		},
	}

	s.ChartRequestVers[specs.shortName()]++
	s.sendJSON(payload)

	internalChannel := make(chan storedCache)
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second)
	defer ctxCancel()

	go func() {
		for {
			select {

			case recvPayload := <-s.TransactionChannel:
				if recvPayload.Chart.RequestID == uniqueID {
					internalChannel <- recvPayload
					return
				}

			case <-ctx.Done():
				break
			}
		}
	}()

	select {

	case recvPayload := <-internalChannel:
		return &recvPayload.Chart, nil

	case <-ctx.Done():
		return nil, ErrNotReceivedInTime
	}

}

// RequestMultipleCharts takes a slice of ChartRequestSignature as an input and
// responds with a a slice of chart objects, it utilizes the cached if it can
// (with updated diffs), or else it makes a new request and waits for it - if a
// ticker does not load in time, ErrNotReceviedInTime is sent as an error
func (s *Session) RequestMultipleCharts(specsSlice []ChartRequestSignature) ([]*ChartStoredCache, []ChartRequestSignature) {

	for {
		if s.MutexLock == false {
			break
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}

	payload := gatewayRequestLoad{}
	response := []*ChartStoredCache{}
	erroredTickers := []ChartRequestSignature{}
	uniqueSpecs := []ChartRequestSignature{}

	for _, spec := range specsSlice {
		// force capitalization of tickers, since the socket is case sensitive
		spec.Ticker = strings.ToUpper(spec.Ticker)

		if s.CurrentState.Chart.RequestID == spec.shortName() {
			response = append(response, &s.CurrentState.Chart)
		}

		spec.UniqueID = fmt.Sprintf("%s-%d", spec.shortName(), s.ChartRequestVers[spec.shortName()])
		uniqueSpecs = append(uniqueSpecs, spec)

		req := gatewayRequest{
			Header: gatewayHeader{
				Service: "chart",
				ID:      spec.UniqueID,
				Ver:     s.ChartRequestVers[spec.shortName()],
			},
			Params: gatewayParams{
				Symbol:            spec.Ticker,
				AggregationPeriod: spec.Width,
				Studies:           []string{},
				Range:             spec.Range,
			},
		}
		s.ChartRequestVers[spec.shortName()]++

		payload.Payload = append(payload.Payload, req)
	}

	s.wsConn.WriteJSON(payload)

	internalChannel := make(chan []*ChartStoredCache)
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second)
	defer ctxCancel()

	go func() {
		for {
			select {

			case recvPayload := <-s.TransactionChannel:
				for index, spec := range uniqueSpecs {
					if recvPayload.Chart.RequestID == spec.UniqueID {
						response = append(response, &recvPayload.Chart)
						uniqueSpecs = append(uniqueSpecs[:index], uniqueSpecs[index+1:]...)
						if len(uniqueSpecs) == 0 {
							internalChannel <- response
						}
						break
					}
				}

			case <-ctx.Done():
				break
			}
		}
	}()

	select {

	case recvPayload := <-internalChannel:
		return recvPayload, erroredTickers

	case <-ctx.Done():
		for _, spec := range uniqueSpecs {
			erroredTickers = append(erroredTickers, spec)
		}
		return response, erroredTickers
	}
}
