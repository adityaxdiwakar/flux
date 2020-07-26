package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Jeffail/gabs"
)

func (s *Session) searchHandler(msg []byte, patch *gabs.Container) {
	patch = patch.S("payloadPatches", "0", "patches", "0")

	var state searchStoredCache
	err := json.Unmarshal(patch.Bytes(), &state)
	if err != nil {
		return
	}

	s.CurrentState.Search = state
	s.TransactionChannel <- s.CurrentState
}

type SearchRequestSignature struct {
	Pattern  string
	Limit    int
	UniqueID string
}

func (r *SearchRequestSignature) shortName() string {
	return fmt.Sprintf("SEARCH#%s@%d", r.Pattern, r.Limit)
}

type searchStoredCache struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value struct {
		Instruments []struct {
			Composite              bool   `json:"composite"`
			Cusip                  string `json:"cusip,omitempty"`
			DaysToExpiration       int    `json:"daysToExpiration"`
			Description            string `json:"description"`
			DisplaySymbol          string `json:"displaySymbol"`
			ExtoEnabled            bool   `json:"extoEnabled"`
			Flags                  int    `json:"flags"`
			FractionalType         string `json:"fractionalType"`
			FutureOption           bool   `json:"futureOption"`
			HasOptions             bool   `json:"hasOptions"`
			ID                     int    `json:"id"`
			Industry               int    `json:"industry"`
			InstrumentType         string `json:"instrumentType"`
			IsFutureProduct        bool   `json:"isFutureProduct"`
			Multiplier             int    `json:"multiplier"`
			RootDisplaySymbol      string `json:"rootDisplaySymbol"`
			RootSymbol             string `json:"rootSymbol"`
			SourceType             string `json:"sourceType"`
			Spc                    int    `json:"spc"`
			SpreadDaysToExpiration string `json:"spreadDaysToExpiration"`
			SpreadsSupported       bool   `json:"spreadsSupported"`
			Symbol                 string `json:"symbol"`
			Tradeable              bool   `json:"tradeable"`
		} `json:"instruments"`
		RequestID  string `json:"requestId"`
		RequestVer int    `json:"requestVer"`
		Service    string `json:"service"`
	} `json:"value"`
}

func (s *Session) RequestSearch(spec SearchRequestSignature) (*searchStoredCache, error) {
	uniqueID := fmt.Sprintf("%s-%d", spec.shortName(), s.SearchRequestVers[spec.shortName()])
	spec.UniqueID = uniqueID

	req := gatewayRequest{
		Service: "instrument_search",
		Limit:   5,
		Pattern: spec.Pattern,
		Ver:     s.SearchRequestVers[spec.shortName()],
		ID:      spec.UniqueID,
	}
	s.SearchRequestVers[spec.shortName()]++

	payload := gatewayRequestLoad{[]gatewayRequest{req}}
	s.wsConn.WriteJSON(payload)

	internalChannel := make(chan storedCache)
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	go func() {
		for {
			select {

			case recvPayload := <-s.TransactionChannel:
				if recvPayload.Search.Value.RequestID == spec.UniqueID {
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
		return &recvPayload.Search, nil

	case <-ctx.Done():
		return nil, ErrNotReceivedInTime

	}

}
