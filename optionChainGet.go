package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
)

// OptionChainGetRequestSignature is the parameter for an option chain request
type OptionChainGetRequestSignature struct {
	// ticker for option chain get request
	Underlying string

	// filter for specific option chain to get
	Filter OptionChainGetFilter

	// internal use only
	UniqueID string
}

// OptionChainGetFilter is the sub-parameter for filtering an option chain
type OptionChainGetFilter struct {
	StrikeQuantity int64    `json:"strikeQuantity,omitempty"`
	SeriesNames    []string `json:"seriesNames,omitempty"`
}

func (o *OptionChainGetRequestSignature) shortName() string {
	return fmt.Sprintf("OPTIONCHAINGET#%s@%v", o.Underlying, o.Filter.SeriesNames)
}

type optionChainPairs struct {
	Strike            float64 `json:"strike"`
	CallSymbol        string  `json:"callSymbol"`
	PutSymbol         string  `json:"putSymbol"`
	CallDisplaySymbol string  `json:"callDisplaySymbol"`
	PutDisplaySymbol  string  `json:"putDisplaySymbol"`
}

// OptionChainSeries are individual series from the option chain get response
type OptionChainSeries struct {
	Expiration       string             `json:"expiration"`
	ExpirationString string             `json:"expirationString"`
	FractionalType   string             `json:"fractionalType"`
	OptionPairs      []optionChainPairs `json:"optionPairs"`
	Spc              float64            `json:"spc"`
	Name             string             `json:"name"`
	Contract         string             `json:"contract"`
	ContractDisplay  string             `json:"contractDisplay"`
	DaysToExpiration int                `json:"daysToExpiration"`
	SettlementType   string             `json:"settlementType"`
}

// OptionChainGetStoredCache is an object containing what is returned from an option chain request
type OptionChainGetStoredCache struct {
	OptionSeries []OptionChainSeries `json:"optionSeries"`
	Service      string              `json:"service"`
	RequestID    string              `json:"requestId"`
	RequestVer   int                 `json:"requestVer"`
}

type optionChainGetWrapper struct {
	Op    string                    `json:"op"`
	Path  string                    `json:"path"`
	Value OptionChainGetStoredCache `json:"value"`
}

// RequestOptionChainGet requests to get an option chain with the input being the OptionChainGetRequestSignature
func (s *Session) RequestOptionChainGet(spec OptionChainGetRequestSignature) (*[]OptionChainSeries, error) {

	uniqueID := fmt.Sprintf("%s-%d", spec.shortName(), s.OptionChainGetRequestVers[spec.shortName()])
	spec.UniqueID = uniqueID

	payload := gatewayRequestLoad{
		Payload: []gatewayRequest{
			{
				Header: gatewayHeader{
					Service: "option_chain/get",
					Ver:     int(s.OptionChainGetRequestVers[spec.shortName()]),
					ID:      spec.UniqueID,
				},
				Params: gatewayParams{
					UnderlyingSymbol: spec.Underlying,
					Filter:           &spec.Filter,
				},
			},
		},
	}

	s.OptionChainGetRequestVers[spec.shortName()]++
	s.sendJSON(payload)

	internalChannel := make(chan storedCache)
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second)
	defer ctxCancel()

	go func() {
		for {
			select {

			case recvPayload := <-s.TransactionChannel:
				if recvPayload.OptionChainGet.RequestID == spec.UniqueID {
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
		return &recvPayload.OptionChainGet.OptionSeries, nil

	case <-ctx.Done():
		return nil, ErrNotReceivedInTime

	}
}

func (s *Session) optionChainGetHandler(msg []byte, patch *gabs.Container) {
	patchBody := patch.S("body", "patches", "0")

	rID := patch.Search("header", "id").String()
	rID = rID[1 : len(rID)-1]

	var state optionChainGetWrapper
	err := json.Unmarshal(patchBody.Bytes(), &state)
	if err != nil {
		return
	}

	state.Value.RequestID = rID
	s.CurrentState.OptionChainGet = state.Value
	s.TransactionChannel <- s.CurrentState
}
