package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
)

// OptionSeriesRequestSignature is the parameter for a chart request
type OptionSeriesRequestSignature struct {
	// ticker for option series request
	Ticker string

	// internal use only
	UniqueID string
}

func (o *OptionSeriesRequestSignature) shortName() string {
	return fmt.Sprintf("OPTIONSERIES#%s", o.Ticker)
}

// OptionSeries is an object containing the data for an option series
type OptionSeries struct {
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

// OptionSeriesCache is an object containing what is returned from an option series request
type OptionSeriesCache struct {
	Series     []OptionSeries `json:"series"`
	Service    string         `json:"service"`
	RequestID  string         `json:"requestId"`
	RequestVer int            `json:"requestVer"`
}

type newOptionSeries struct {
	Op    string            `json:"op"`
	Path  string            `json:"path"`
	Value OptionSeriesCache `json:"value"`
}

// RequestOptionSeries returns options series data for a specific series based on the spec provided
func (s *Session) RequestOptionSeries(spec OptionSeriesRequestSignature) (*[]OptionSeries, error) {

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
		if len(recvPayload.OptionSeries.Series) == 0 {
			return nil, ErrNotReceivedInTime
		}
		return &recvPayload.OptionSeries.Series, nil

	case <-ctx.Done():
		return nil, ErrNotReceivedInTime
	}
}

func (s *Session) optionSeriesHandler(msg []byte, patch *gabs.Container) {
	patchBody := patch.S("body", "patches", "0")

	rID := patch.Search("header", "id").String()
	rID = rID[1 : len(rID)-1]

	var state newOptionSeries
	err := json.Unmarshal(patchBody.Bytes(), &state)
	if err != nil {
		return
	}

	state.Value.RequestID = rID
	s.CurrentState.OptionSeries = state.Value
	s.TransactionChannel <- s.CurrentState
}
