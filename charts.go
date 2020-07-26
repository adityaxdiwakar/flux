package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

type chartStoredCache struct {
	Symbol     string `json:"symbol"`
	Instrument struct {
		Symbol                 string `json:"symbol"`
		RootSymbol             string `json:"rootSymbol"`
		DisplaySymbol          string `json:"displaySymbol"`
		RootDisplaySymbol      string `json:"rootDisplaySymbol"`
		FutureOption           bool   `json:"futureOption"`
		Description            string `json:"description"`
		Multiplier             int    `json:"multiplier"`
		SpreadsSupported       bool   `json:"spreadsSupported"`
		Tradeable              bool   `json:"tradeable"`
		InstrumentType         string `json:"instrumentType"`
		ID                     int    `json:"id"`
		SourceType             string `json:"sourceType"`
		IsFutureProduct        bool   `json:"isFutureProduct"`
		HasOptions             bool   `json:"hasOptions"`
		Composite              bool   `json:"composite"`
		FractionalType         string `json:"fractionalType"`
		DaysToExpiration       int    `json:"daysToExpiration"`
		SpreadDaysToExpiration string `json:"spreadDaysToExpiration"`
		Cusip                  string `json:"cusip"`
		Industry               int    `json:"industry"`
		Spc                    int    `json:"spc"`
		ExtoEnabled            bool   `json:"extoEnabled"`
		Flags                  int    `json:"flags"`
	} `json:"instrument"`
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
	Value chartStoredCache `json:"value"`
}

type updateChartObject struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value int    `json:"value"`
}

func (s *Session) chartHandler(msg []byte, gab *gabs.Container) {
	patches, err := gab.S("payloadPatches").Children()
	if err != nil {
		return
	}
	for _, patch := range patches {
		patch = patch.S("patches", "0")

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
			modifiedChart, err = json.Marshal([]newChartObject{newChart})
		} else {
			updatedChart := updateChartObject{}
			json.Unmarshal(bytesJson, &updatedChart)
			updatedChart.Path = "/chart" + updatedChart.Path
			modifiedChart, err = json.Marshal([]updateChartObject{updatedChart})
		}

		if err != nil {
			log.Fatal(err)
		}
		jspatch, err := jsonpatch.DecodePatch(modifiedChart)
		if err != nil {
			log.Fatal(err)
		}

		byteState, _ := json.Marshal(s.CurrentState)

		byteState, err = jspatch.Apply(byteState)
		if err != nil {
			log.Fatal(err)
		}

		var newState storedCache
		json.Unmarshal(byteState, &newState)

		if patch.S("path").String() == `""` {
			s.TransactionChannel <- newState
		}
	}
}

// RequestChart takes a ChartRequestSignature as an input and responds with a
// chartStoredCache object, it utilizes the cached if it can (with updated diffs), or
// else it makes a new request and waits for it - if a ticker does not load in
// time, ErrNotReceviedInTime is sent as an error
func (s *Session) RequestChart(specs ChartRequestSignature) (*chartStoredCache, error) {

	// force capitalization of tickers, since the socket is case sensitive
	specs.Ticker = strings.ToUpper(specs.Ticker)

	if s.CurrentState.Chart.RequestID == specs.shortName() {
		return &s.CurrentState.Chart, nil
	}

	uniqueID := fmt.Sprintf("%s-%d", specs.shortName(), s.ChartRequestVers[specs.shortName()])

	req := gatewayRequest{
		Service:           "chart",
		ID:                uniqueID,
		Ver:               s.ChartRequestVers[specs.shortName()],
		Symbol:            specs.Ticker,
		AggregationPeriod: specs.Width,
		Studies:           []string{},
		Range:             specs.Range,
	}
	s.ChartRequestVers[specs.shortName()]++
	payload := gatewayRequestLoad{[]gatewayRequest{req}}
	s.wsConn.WriteJSON(payload)

	internalChannel := make(chan storedCache)
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

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

	//unreachable code
	return nil, nil
}

// RequestMultipleCharts takes a slice of ChartRequestSignature as an input and
// responds with a a slice of chart objects, it utilizes the cached if it can
// (with updated diffs), or else it makes a new request and waits for it - if a
// ticker does not load in time, ErrNotReceviedInTime is sent as an error
func (s *Session) RequestMultipleCharts(specsSlice []ChartRequestSignature) ([]*chartStoredCache, []ChartRequestSignature) {

	payload := gatewayRequestLoad{}
	response := []*chartStoredCache{}
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
			Service:           "chart",
			ID:                spec.UniqueID,
			Ver:               s.ChartRequestVers[spec.shortName()],
			Symbol:            spec.Ticker,
			AggregationPeriod: spec.Width,
			Studies:           []string{},
			Range:             spec.Range,
		}
		s.ChartRequestVers[spec.shortName()]++

		payload.Payload = append(payload.Payload, req)
	}

	s.wsConn.WriteJSON(payload)

	internalChannel := make(chan []*chartStoredCache)
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

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
