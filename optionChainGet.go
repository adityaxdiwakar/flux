package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
)

type OptionChainGetRequestSignature struct {
	// ticker for option chain get request
	Underlying string

	// filter for specific option chain to get
	Filter OptionChainGetFilter

	// internal use only
	UniqueID string
}

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

type optionChainSeries struct {
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

type optionChainGetStoredCache struct {
	OptionSeries []optionChainSeries `json:"optionSeries"`
	Service      string              `json:"service"`
	RequestID    string              `json:"requestId"`
	RequestVer   int                 `json:"requestVer"`
}

type optionChainGetWrapper struct {
	Op    string                    `json:"op"`
	Path  string                    `json:"path"`
	Value optionChainGetStoredCache `json:"value"`
}

func (s *Session) RequestOptionChainGet(spec OptionChainGetRequestSignature) (*optionChainGetStoredCache, error) {

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
		return &recvPayload.OptionChainGet, nil

	case <-ctx.Done():
		return nil, ErrNotReceivedInTime

	}
}

func (s *Session) optionChainGetHandler(msg []byte, patch *gabs.Container) {
	patch = patch.S("payloadPatches", "0", "patches", "0")

	var state optionChainGetWrapper
	err := json.Unmarshal(patch.Bytes(), &state)
	if err != nil {
		return
	}

	state.Path = "/optionChainGet"
	s.CurrentState.OptionChainGet = state.Value
	s.TransactionChannel <- s.CurrentState
}
