package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
	jsonpatch "github.com/evanphx/json-patch"
)

const (
	// ProbabilityITM field
	ProbabilityITM = QuoteField("PROBABILITY_ITM")
	// OpenInterest field
	OpenInterest = QuoteField("OPEN_INT")
)

// OptionQuoteRequestSignature is the parameter for an option quote request
type OptionQuoteRequestSignature struct {
	// ticker for option chain get request
	Underlying string

	Exchange string

	Fields []QuoteField

	// filter for specific option chain to get
	Filter OptionQuoteFilter

	// internal use only
	UniqueID string
}

// OptionQuoteFilter is the sub-parameter for filtering an option quote
type OptionQuoteFilter struct {
	SeriesNames []string `json:"seriesNames"`
	MaxStrike   float64  `json:"maxStrike"`
	MinStrike   float64  `json:"minStrike"`
}

func (o *OptionQuoteRequestSignature) shortName() string {
	return fmt.Sprintf("OPTIONCHAINGET#%s@%v", o.Underlying, o.Filter.SeriesNames)
}

// OptionQuoteCache is an object containing what is returned from an option quote request
type OptionQuoteCache struct {
	Items      []OptionQuoteItem `json:"items"`
	Exchanges  []string          `json:"exchanges"`
	Service    string            `json:"service"`
	RequestID  string            `json:"requestId"`
	RequestVer int               `json:"requestVer"`
}

// OptionQuoteItem is a single option quote
type OptionQuoteItem struct {
	Symbol string `json:"symbol"`
	Values struct {
		ASK            float64 `json:"ASK"`
		BID            float64 `json:"BID"`
		DELTA          float64 `json:"DELTA"`
		OPENINT        float64 `json:"OPEN_INT"`
		PROBABILITYITM float64 `json:"PROBABILITY_ITM"`
		VOLUME         int     `json:"VOLUME"`
	} `json:"values"`
}

type newOptionQuote struct {
	Op    string           `json:"op"`
	Path  string           `json:"path"`
	Value OptionQuoteCache `json:"value"`
}

// RequestOptionQuote requests to get an option quote with the spec OptionQuoteRequestSignature
func (s *Session) RequestOptionQuote(spec OptionQuoteRequestSignature) (*OptionQuoteCache, error) {

	uniqueID := fmt.Sprintf("%s-%d", spec.shortName(), s.OptionQuoteRequestVers[spec.shortName()])
	spec.UniqueID = uniqueID

	payload := gatewayRequestLoad{
		Payload: []gatewayRequest{
			{
				Header: gatewayHeader{
					Service: "quotes/options",
					Ver:     int(s.OptionQuoteRequestVers[spec.shortName()]),
					ID:      spec.UniqueID,
				},
				Params: gatewayParams{
					UnderlyingSymbol: spec.Underlying,
					Exchange:         spec.Exchange,
					Filter:           &spec.Filter,
					QuoteFields:      spec.Fields,
				},
			},
		},
	}

	s.OptionQuoteRequestVers[spec.shortName()]++
	s.sendJSON(payload)

	internalChannel := make(chan storedCache)
	ctx, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCancel()

	go func() {
		for {
			select {

			case recvPayload := <-s.TransactionChannel:
				if recvPayload.OptionQuote.RequestID == spec.UniqueID {
					time.Sleep(1000 * time.Millisecond)
					internalChannel <- s.CurrentState
					return
				}

			case <-ctx.Done():
				return

			}

		}
	}()

	select {

	case recvPayload := <-internalChannel:
		return &recvPayload.OptionQuote, nil

	case <-ctx.Done():
		return nil, ErrNotReceivedInTime

	}
}

func (s *Session) optionQuoteHandler(msg []byte, patch *gabs.Container) {

	rID := patch.Search("header", "id").String()
	rID = rID[1 : len(rID)-1]
	patch = patch.S("body", "patches", "0")

	newState := s.CurrentState
	path := patch.S("path").String()
	if path == "\"\"" {
		newState.OptionQuote = OptionQuoteCache{}
	}
	patch.Set(fmt.Sprintf("/optionQuote%s", path[1:len(path)-1]), "path")

	bytesJSON, _ := patch.MarshalJSON()
	patchStr := "[" + string(bytesJSON) + "]"
	jspatch, err := jsonpatch.DecodePatch([]byte(patchStr))

	if err != nil {
		return
	}

	byteState, _ := json.Marshal(newState)

	byteState, err = jspatch.Apply(byteState)
	if err != nil {
		return
	}

	json.Unmarshal(byteState, &newState)
	newState.OptionQuote.RequestID = rID
	s.CurrentState = newState
	s.TransactionChannel <- s.CurrentState
}
