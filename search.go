package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
)

func (s *Session) searchHandler(msg []byte, patch *gabs.Container) {
	patch = patch.S("payloadPatches", "0", "patches", "0")

	var state searchStoredCache
	err := json.Unmarshal(patch.Bytes(), &state)
	if err != nil {
		return
	}

	state.Path = "/instrument_search"
	s.CurrentState.Search = state
	s.TransactionChannel <- s.CurrentState
}

// SearchRequest Signature is the parameter for a search request
type SearchRequestSignature struct {
	// pattern is the search query for the request
	Pattern string

	// limit is the number of results to pull up, typically 3-5 is normal
	Limit int

	// internal use only
	UniqueID string
}

// shortname presented as SEARCH#PATTERN@LIMIT
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

// RequestSearch takes a SearchRequestSignature as an input and responds with a
// search query, it does not utilize a cache (although it maintains one) so the
// query made will be up to date with the servers. If the query is not loaded
// within a certain time, ErrNotReceivedInTime is sent as an error
func (s *Session) RequestSearch(spec SearchRequestSignature) (*searchStoredCache, error) {

	uniqueID := fmt.Sprintf("%s-%d", spec.shortName(), s.SearchRequestVers[spec.shortName()])
	spec.UniqueID = uniqueID

	payload := gatewayRequestLoad{
		Payload: []gatewayRequest{
			{
				Header: gatewayHeader{
					Service: "instrument_search",
					Ver:     s.SearchRequestVers[spec.shortName()],
					ID:      spec.UniqueID,
				},
				Params: gatewayParams{
					Limit:   5,
					Pattern: spec.Pattern,
				},
			},
		},
	}

	s.SearchRequestVers[spec.shortName()]++
	s.sendJSON(payload)

	internalChannel := make(chan storedCache)
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second)
	defer ctxCancel()

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
