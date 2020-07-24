package flux

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	jsonpatch "github.com/evanphx/json-patch"
)

func (s *Session) chartHandler(msg []byte, gab *gabs.Container) {
	for _, patch := range gab.S("payloadPatches", "0", "patches").Children() {

		var err error
		bytesJson, err := patch.MarshalJSON()
		if err != nil {
			log.Fatal(err)
		}

		var modifiedChart []byte

		if patch.S("path").String() == "/error" {
			return
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

func (s *Session) RequestChart(specs ChartRequestSignature) (*CachedData, error) {
	// prepare request for gateway transaction

	s.TransactionChannel = make(chan CachedData)

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

	internalChannel := make(chan CachedData)
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
		return nil, errors.New("error: took too long to respond, try again")

	}

	//unreachable code
	return nil, nil
}
