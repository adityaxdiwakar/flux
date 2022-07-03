package flux

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func (s *Session) searchHandler(data Payload) {
	route, ok := s.SearchRouteTable[data.Header.ID]
	if !ok {
		// Receiving a response for an ID that isn't in the route table
		// probably means the request context timed out before receiving
		// this, so we can ignore this.
		return
	}
	resp := RespTuple[SearchData]{}

	if data.Header.Type == "error" {
		body := ErroredBody{}
		if err := json.Unmarshal(data.Body, &body); err != nil {
			resp.Err = errors.New("Received an errored response that could not be parsed")
		} else {
			resp.Err = errors.New(body.Message)
		}
		route <- resp
		return
	}

	if data.Header.Type == "snapshot" {
		body := SearchData{}
		if err := json.Unmarshal(data.Body, &body); err != nil {
			resp.Err = errors.New("Received a snapshot response that could not be parsed")
			route <- resp
			return
		}
		body.RequestID = data.Header.ID
		body.RequestVer = data.Header.Ver
		s.Cache.Search = body

		resp.Body = s.Cache.Search
		resp.Err = nil
		route <- resp

		return
	}

	if data.Header.Type != "patch" {
		resp.Err = errors.New("Response body cannot be understood: neither error nor patch")
		route <- resp
		return
	}

	body := Body[SearchData]{}
	if err := json.Unmarshal(data.Body, &body); err != nil {
		resp.Err = errors.New("Response body could not be interpreted as type Body")
		route <- resp
		return
	}

	newState := s.Cache
	for idx := 0; idx < len(body.Patches); idx++ {
		// TODO: handle the "error" path case (might be deprecated)

		if body.Patches[idx].Path == "" {
			newState.Search = SearchData{}
		}
		body.Patches[idx].Path = "/search" + body.Patches[idx].Path
	}

	applyPatch(body, &newState)

	s.Cache = newState
	s.Cache.Search.RequestID = data.Header.ID
	s.Cache.Search.RequestVer = data.Header.Ver

	resp.Body = s.Cache.Search
	resp.Err = nil
	route <- resp
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
type SearchData struct {
	Instruments []struct {
		Description   string `json:"description"`
		DisplaySymbol string `json:"displaySymbol"`
		Symbol        string `json:"symbol"`
	} `json:"instruments"`
	RequestID  string `json:"requestId"`
	RequestVer int    `json:"requestVer"`
}

// RequestSearch takes a SearchRequestSignature as an input and responds with a
// search query, it does not utilize a cache (although it maintains one) so the
// query made will be up to date with the servers. If the query is not loaded
// within a certain time, ErrNotReceivedInTime is sent as an error
func (s *Session) RequestSearch(spec SearchRequestSignature) (*SearchData, error) {

	uniqueID := fmt.Sprintf("%s-%d", spec.shortName(),
		s.SearchRequestVers[spec.shortName()])
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
	s.SearchRouteTable[uniqueID] = make(chan RespTuple[SearchData])
	s.sendJSON(payload)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	select {

	case recvPayload := <-s.SearchRouteTable[uniqueID]:
		return &recvPayload.Body, recvPayload.Err

	case <-ctx.Done():
		return nil, ErrNotReceivedInTime
	}

}
