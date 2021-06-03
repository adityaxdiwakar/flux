package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
	jsonpatch "github.com/evanphx/json-patch"
)

func (s *Session) searchHandler(msg []byte, patch *gabs.Container) {
	rID := patch.Search("header", "id").String()
	rID = rID[1 : len(rID)-1]
	rService := patch.Search("header", "service").String()
	rService = rService[1 : len(rService)-1]

	patch = patch.S("body", "patches", "0")

	newSearch := newSearchObject{}
	json.Unmarshal(patch.Bytes(), &newSearch)
	newSearch.Path = "/search"
	newSearch.Value.RequestID = rID
	newSearch.Value.Service = rService
	modifiedSearch, err := json.Marshal([]newSearchObject{newSearch})
	if err != nil {
		return
	}

	jspatch, err := jsonpatch.DecodePatch(modifiedSearch)
	if err != nil {
		return
	}

	byteState, _ := json.Marshal(s.CurrentState)

	byteState, err = jspatch.Apply(byteState)
	if err != nil {
		return
	}

	var newState storedCache
	json.Unmarshal(byteState, &newState)

	s.CurrentState = newState
	s.TransactionChannel <- s.CurrentState
}

// SearchRequestSignature is the parameter for a search request
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

// SearchStoredCache is the response for search request
type SearchStoredCache struct {
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
}

type newSearchObject struct {
	Op    string            `json:"op"`
	Path  string            `json:"path"`
	Value SearchStoredCache `json:"value"`
}

// RequestSearch takes a SearchRequestSignature as an input and responds with a
// search query, it does not utilize a cache (although it maintains one) so the
// query made will be up to date with the servers. If the query is not loaded
// within a certain time, ErrNotReceivedInTime is sent as an error
func (s *Session) RequestSearch(spec SearchRequestSignature) (*SearchStoredCache, error) {

	uniqueID := fmt.Sprintf("%s-%d", spec.shortName(), s.SearchRequestVers[spec.shortName()])
	spec.UniqueID = uniqueID

	payload := gatewayRequestLoad{
		Payload: []gatewayRequest{
			{
				Header: gatewayHeader{
					Service: "instrument_search",
					Ver:     int(s.SearchRequestVers[spec.shortName()]),
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
				if recvPayload.Search.RequestID == spec.UniqueID {
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
