package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
)

type OptionSeriesRequestSignature struct {
	// ticker for option series request
	Ticker string

	// internal use only
	UniqueID string
}

func (o *OptionSeriesRequestSignature) shortName() string {
	return fmt.Sprintf("OPTIONSERIES#%s", o.Ticker)
}

type optionSeries struct {
	Underlying      string    `json:"underlying"`
	Name            string    `json:"name"`
	Spc             float64   `json:"spc"`
	Multiplier      float64   `json:"multiplier"`
	ExpirationStyle string    `json:"expirationStyle"`
	IsEuropean      bool      `json:"isEuropean"`
	Expiration      time.Time `json:"expiration"`
	LastTradeDate   time.Time `json:"lastTradeDate"`
	SettlementType  string    `json:"settlementType"`
}

type optionSeriesValue struct {
	Series     []optionSeries `json:"series"`
	Service    string         `json:"service"`
	RequestID  string         `json:"requestId"`
	RequestVer int            `json:"requestVer"`
}

type optionSeriesStoredCache struct {
	Op    string            `json:"op"`
	Path  string            `json:"path"`
	Value optionSeriesValue `json:"value"`
}

func (s *Session) RequestOptionSeries(spec OptionSeriesRequestSignature) (*optionSeriesValue, error) {

	uniqueID := fmt.Sprintf("%s-%d", spec.shortName(), s.OptionSeriesRequestVers[spec.shortName()])
	spec.UniqueID = uniqueID

	payload := gatewayRequestLoad{
		Payload: []gatewayRequest{
			{
				Header: gatewayHeader{
					Service: "optionSeries",
					Ver:     int(s.OptionSeriesRequestVers[spec.shortName()]),
					ID:      spec.UniqueID,
				},
				Params: gatewayParams{
					Underlying: spec.Ticker,
				},
			},
		},
	}

	s.OptionSeriesRequestVers[spec.shortName()]++
	s.sendJSON(payload)

	internalChannel := make(chan storedCache)
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second)
	defer ctxCancel()

	go func() {
		for {
			select {

			case recvPayload := <-s.TransactionChannel:
				if recvPayload.OptionSeries.RequestID == spec.UniqueID {
					internalChannel <- recvPayload
					return
				}

			case <-ctx.Done():
				return

			}
		}
	}()

	select {

	case recvPayload := <-internalChannel:
		return &recvPayload.OptionSeries, nil

	case <-ctx.Done():
		return nil, ErrNotReceivedInTime
	}
}

func (s *Session) optionSeriesHandler(msg []byte, patch *gabs.Container) {
	patch = patch.S("payloadPatches", "0", "patches", "0")

	var state optionSeriesStoredCache
	err := json.Unmarshal(patch.Bytes(), &state)
	if err != nil {
		return
	}

	state.Path = "/optionSeries"
	s.CurrentState.OptionSeries = state.Value
	s.TransactionChannel <- s.CurrentState
}
