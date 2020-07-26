package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
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

type storedCache struct {
	Data chartStoredCache `json:"data"`
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

	for _, patch := range gab.S("payloadPatches").Children() {
		patch = patch.S("patches", "0")

		// TODO: implement actual error handling, currently using log.Fatal()
		// which is bad
		var err error
		bytesJson, err := patch.MarshalJSON()
		if err != nil {
			log.Fatal(err)
		}

		var modifiedChart []byte

		if patch.S("path").String() == "/error" {
			continue
		}

		if patch.S("path").String() == `""` {
			newChart := newChartObject{}
			json.Unmarshal(bytesJson, &newChart)
			newChart.Path = "/data"
			modifiedChart, err = json.Marshal([]newChartObject{newChart})
		} else {
			updatedChart := updateChartObject{}
			json.Unmarshal(bytesJson, &updatedChart)
			updatedChart.Path = "/data" + updatedChart.Path
			modifiedChart, err = json.Marshal([]updateChartObject{updatedChart})
		}

		if err != nil {
			log.Fatal(err)
		}
		jspatch, err := jsonpatch.DecodePatch(modifiedChart)
		if err != nil {
			log.Fatal(err)
		}

		s.CurrentState, err = jspatch.Apply(s.CurrentState)
		if err != nil {
			log.Fatal(err)
		}

		if patch.S("path").String() == `""` {
			d, _ := s.dataAsChartObject()

			s.CurrentChartHash = d.RequestID
			s.TransactionChannel <- *d
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

	if s.CurrentChartHash == specs.shortName() {
		d, err := s.dataAsChartObject()
		if err != nil {
			return nil, err
		}
		return d, nil
	}

	uniqueID := fmt.Sprintf("%s-%d", specs.shortName(), s.RequestVers[specs.shortName()])

	req := gatewayRequest{
		Service:           "chart",
		ID:                uniqueID,
		Ver:               s.RequestVers[specs.shortName()],
		Symbol:            specs.Ticker,
		AggregationPeriod: specs.Width,
		Studies:           []string{},
		Range:             specs.Range,
	}
	s.RequestVers[specs.shortName()]++
	payload := gatewayRequestLoad{[]gatewayRequest{req}}
	s.wsConn.WriteJSON(payload)

	internalChannel := make(chan chartStoredCache)
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	go func() {
		for {
			select {

			case recvPayload := <-s.TransactionChannel:
				if recvPayload.RequestID == uniqueID {
					internalChannel <- recvPayload
					break
				}

			case <-ctx.Done():
				break
			}
		}
	}()

	select {

	case recvPayload := <-internalChannel:
		return &recvPayload, nil

	case <-ctx.Done():
		return nil, ErrNotReceivedInTime
	}

	//unreachable code
	return nil, nil
}

// RequestMultipleCharts takes a slice of ChartRequestSignature as an input and responds with a
// chartStoredCache object, it utilizes the cached if it can (with updated diffs), or
// else it makes a new request and waits for it - if a ticker does not load in
// time, ErrNotReceviedInTime is sent as an error
func (s *Session) RequestMultipleCharts(specsSlice []ChartRequestSignature) ([]*chartStoredCache, []ChartRequestSignature) {

	payload := gatewayRequestLoad{}
	response := []*chartStoredCache{}
	erroredTickers := []ChartRequestSignature{}
	uniqueSpecs := []ChartRequestSignature{}

	for _, spec := range specsSlice {
		// force capitalization of tickers, since the socket is case sensitive
		spec.Ticker = strings.ToUpper(spec.Ticker)

		if s.CurrentChartHash == spec.shortName() {
			d, err := s.dataAsChartObject()
			if err != nil {
				erroredTickers = append(erroredTickers, spec)
			}
			response = append(response, d)
		}

		spec.UniqueID = fmt.Sprintf("%s-%d", spec.shortName(), s.RequestVers[spec.shortName()])
		uniqueSpecs = append(uniqueSpecs, spec)

		req := gatewayRequest{
			Service:           "chart",
			ID:                spec.UniqueID,
			Ver:               s.RequestVers[spec.shortName()],
			Symbol:            spec.Ticker,
			AggregationPeriod: spec.Width,
			Studies:           []string{},
			Range:             spec.Range,
		}
		s.RequestVers[spec.shortName()]++

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
					if recvPayload.RequestID == spec.UniqueID {
						response = append(response, &recvPayload)
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
