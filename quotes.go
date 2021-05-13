package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	jsonpatch "github.com/evanphx/json-patch"
)

type QuoteField string

const (
	Mark              = QuoteField("MARK")
	MarkChange        = QuoteField("MARK_CHANGE")
	MarkPercentChange = QuoteField("MARK_PERCENT_CHANGE")
	NetChange         = QuoteField("NET_CHANGE")
	NetPercentChange  = QuoteField("NET_CHANGE_PERCENT")
	Bid               = QuoteField("BID")
	BidExchange       = QuoteField("BID_EXCHANGE")
	Ask               = QuoteField("ASK")
	AskExchange       = QuoteField("ASK_EXCHANGE")
	BidSize           = QuoteField("BID_SIZE")
	AskSize           = QuoteField("ASK_SIZE")
	Volume            = QuoteField("VOLUME")
	Open              = QuoteField("OPEN")
	High52            = QuoteField("HIGH52")
	Low52             = QuoteField("LOW52")
	High              = QuoteField("HIGH")
	Low               = QuoteField("LOW")
	VWAP              = QuoteField("VWAP")
	VolIdx            = QuoteField("VOLATILITY_INDEX")
	ImplVol           = QuoteField("IMPLIED_VOLATILITY")
	MMMove            = QuoteField("MARKET_MAKER_MOVE")
	PercentIV         = QuoteField("PERCENT_IV")
	HistoricalVol30   = QuoteField("HISTORICAL_VOLATILITY_30_DAYS")
	MktCap            = QuoteField("MARKET_CAP")
	Beta              = QuoteField("BETA")
	PE                = QuoteField("PE")
	InitMargin        = QuoteField("INITIAL_MARGIN")
	Last              = QuoteField("LAST")
	LastSize          = QuoteField("LAST_SIZE")
	LastExchange      = QuoteField("LAST_EXCHANGE")
	Rho               = QuoteField("RHO")
	BorrowStatus      = QuoteField("BORROW_STATUS")
	Delta             = QuoteField("DELTA")
	Gamma             = QuoteField("GAMMA")
	Theta             = QuoteField("THETA")
	Vega              = QuoteField("VEGA")
	DivAmount         = QuoteField("DIV_AMOUNT")
	EPS               = QuoteField("EPS")
	ExdDivDate        = QuoteField("EXD_DIV_DATE")
	Yield             = QuoteField("YIELD")
	FrontVol          = QuoteField("FRONT_VOLATILITY")
	BackVol           = QuoteField("BACK_VOLATILITY")
	VolDiff           = QuoteField("VOLATILITY_DIFFERENCE")
	Close             = QuoteField("CLOSE")
)

type QuoteRequestSignature struct {
	// ticker for the content of a quote request
	Ticker string

	// how frequently to get a quote (TODO: useless for flux)
	RefreshRate int

	// field properties (detailed in documentation)

	Fields []QuoteField
}

func (q *QuoteRequestSignature) shortName() string {
	return fmt.Sprintf("QUOTE#%s@%d(W:%d)", q.Ticker, q.RefreshRate, len(q.Fields))
}

type QuoteStoredCache struct {
	Items []struct {
		Symbol string `json:"symbol"`
		Values struct {
			ASK                        float64 `json:"ASK,omitempty"`
			ASKEXCHANGE                string  `json:"ASK_EXCHANGE,omitempty"`
			ASKSIZE                    int     `json:"ASK_SIZE,omitempty"`
			BACKVOLATILITY             float64 `json:"BACK_VOLATILITY,omitempty"`
			BETA                       float64 `json:"BETA,omitempty"`
			BID                        float64 `json:"BID,omitempty"`
			BIDEXCHANGE                string  `json:"BID_EXCHANGE,omitempty"`
			BIDSIZE                    int     `json:"BID_SIZE,omitempty"`
			BORROWSTATUS               string  `json:"BORROW_STATUS,omitempty"`
			CLOSE                      float64 `json:"CLOSE,omitempty"`
			DELTA                      float64 `json:"DELTA,omitempty"`
			FRONTVOLATILITY            float64 `json:"FRONT_VOLATILITY,omitempty"`
			GAMMA                      float64 `json:"GAMMA,omitempty"`
			HIGH                       float64 `json:"HIGH,omitempty"`
			HISTORICALVOLATILITY30DAYS float64 `json:"HISTORICAL_VOLATILITY_30_DAYS,omitempty"`
			IMPLIEDVOLATILITY          float64 `json:"IMPLIED_VOLATILITY,omitempty"`
			INITIALMARGIN              float64 `json:"INITIAL_MARGIN,omitempty"`
			LAST                       float64 `json:"LAST,omitempty"`
			LASTEXCHANGE               string  `json:"LAST_EXCHANGE,omitempty"`
			LASTSIZE                   int     `json:"LAST_SIZE,omitempty"`
			LOW                        float64 `json:"LOW,omitempty"`
			MARK                       float64 `json:"MARK,omitempty"`
			MARKETCAP                  int     `json:"MARKET_CAP,omitempty"`
			MARKCHANGE                 float64 `json:"MARK_CHANGE,omitempty"`
			MARKPERCENTCHANGE          float64 `json:"MARK_PERCENT_CHANGE,omitempty"`
			NETCHANGE                  float64 `json:"NET_CHANGE,omitempty"`
			NETCHANGEPERCENT           float64 `json:"NET_CHANGE_PERCENT,omitempty"`
			OPEN                       float64 `json:"OPEN,omitempty"`
			PERCENTILEIV               float64 `json:"PERCENTILE_IV,omitempty"`
			RHO                        float64 `json:"RHO,omitempty"`
			THETA                      float64 `json:"THETA,omitempty"`
			VEGA                       float64 `json:"VEGA,omitempty"`
			VOLATILITYDIFFERENCE       float64 `json:"VOLATILITY_DIFFERENCE,omitempty"`
			VOLATILITYINDEX            float64 `json:"VOLATILITY_INDEX,omitempty"`
			VOLUME                     int     `json:"VOLUME,omitempty"`
			VWAP                       float64 `json:"VWAP,omitempty"`
		} `json:"values"`
	} `json:"items"`
	Service   string `json:"service"`
	RequestID string `json:"requestId"`
}

type newQuoteObject struct {
	Op    string           `json:"op"`
	Path  string           `json:"path"`
	Value QuoteStoredCache `json:"value"`
}

type updatedQuoteObject struct {
	Op    string  `json:"op"`
	Path  string  `json:"path"`
	Value float64 `json:"value"`
}

func (s *Session) quoteHandler(msg []byte, gab *gabs.Container) {
	// get the request id from the header
	rID := gab.Search("header", "id").String()
	rID = rID[1 : len(rID)-1]
	rService := gab.Search("header", "service").String()
	rService = rService[1 : len(rService)-1]

	// get the patches
	patches := gab.S("body", "patches").Children()
	for _, patch := range patches {
		// TODO: implement actual error handling, currently using log.Fatal()
		// which is bad
		var err error
		bytesJson := patch.Bytes()

		var modifiedQuote []byte

		if patch.S("path").String() == "/error" {
			continue
		}

		if patch.S("path").String() == `""` {
			newQuote := newQuoteObject{}
			json.Unmarshal(bytesJson, &newQuote)
			newQuote.Path = "/quote"
			newQuote.Value.RequestID = rID
			newQuote.Value.Service = rService
			modifiedQuote, err = json.Marshal([]newQuoteObject{newQuote})
		} else {
			if rID != fmt.Sprintf("\"%s\"", s.CurrentState.Quote.RequestID) {
				continue
			}
			updatedQuote := updatedQuoteObject{}
			json.Unmarshal(bytesJson, &updatedQuote)
			updatedQuote.Path = "/quote" + updatedQuote.Path
			modifiedQuote, err = json.Marshal([]updatedQuoteObject{updatedQuote})
		}

		if err != nil {
			return
		}

		jspatch, err := jsonpatch.DecodePatch(modifiedQuote)
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

		if patch.S("path").String() == `""` {
			s.TransactionChannel <- newState
		}
	}
}

func (s *Session) RequestQuote(specs QuoteRequestSignature) (*QuoteStoredCache, error) {

	// force capitalization of tickers, since the socket is case sensitive
	specs.Ticker = strings.ToUpper(specs.Ticker)

	// TODO: make this functional, it will currently not work
	if s.CurrentState.Chart.RequestID == specs.shortName() {
		return &s.CurrentState.Quote, nil
	}

	uniqueID := fmt.Sprintf("%s-%d", specs.shortName(), s.QuoteRequestVers[specs.shortName()])

	payload := gatewayRequestLoad{
		Payload: []gatewayRequest{
			{
				Header: gatewayHeader{
					Service: "quotes",
					ID:      uniqueID,
					Ver:     s.QuoteRequestVers[specs.shortName()],
				},
				Params: gatewayParams{
					Account:     "COMBINED ACCOUNT",
					Symbols:     []string{specs.Ticker},
					RefreshRate: specs.RefreshRate,
					QuoteFields: specs.Fields,
				},
			},
		},
	}

	s.QuoteRequestVers[specs.shortName()]++
	s.sendJSON(payload)

	internalChannel := make(chan storedCache)
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second)
	defer ctxCancel()

	go func() {
		for {
			select {

			case recvPayload := <-s.TransactionChannel:
				if recvPayload.Quote.RequestID == uniqueID {
					internalChannel <- recvPayload
					return
				}

			case <-ctx.Done():
				break
			}
		}
	}()

	select {

	case recvPayload := <-internalChannel:
		return &recvPayload.Quote, nil

	case <-ctx.Done():
		return nil, ErrNotReceivedInTime
	}
}
