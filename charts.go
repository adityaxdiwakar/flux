package flux

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	jsonpatch "github.com/evanphx/json-patch"
)

func (s *Session) chartHandler(msg []byte, gab *gabs.Container) {
	for _, patch := range gab.S("payloadPatches", "0", "patches").Children() {

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
			s.TransactionChannel <- *d
		}
	}
}

// RequestChart takes a ChartRequestSignature as an input and responds with a
// cachedData object, it utilizes the cached if it can (with updated diffs), or
// else it makes a new request and waits for it - if a ticker does not load in
// time, ErrNotReceviedInTime is sent as an error
func (s *Session) RequestChart(specs ChartRequestSignature) (*cachedData, error) {
	// prepare request for gateway transaction

	s.TransactionChannel = make(chan cachedData)

	if s.CurrentChartHash == specs.shortName() {
		d, err := s.dataAsChartObject()
		if err != nil {
			return nil, err
		}
		return d, nil
	}

	s.CurrentChartHash = specs.shortName()

	req := gatewayRequest{
		Service:           "chart",
		ID:                "chart",
		Ver:               0,
		Symbol:            specs.Ticker,
		AggregationPeriod: specs.Width,
		Studies:           []string{},
		Range:             specs.Range,
	}
	payload := gatewayRequestLoad{[]gatewayRequest{req}}
	s.wsConn.WriteJSON(payload)

	internalChannel := make(chan cachedData)
	ctx, _ := context.WithTimeout(context.Background(), 750*time.Millisecond)

	go func() {
		for {
			select {

			case recvPayload := <-s.TransactionChannel:
				if strings.ToUpper(recvPayload.Symbol) == strings.ToUpper(specs.Ticker) {
					internalChannel <- recvPayload
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
