package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	jsonpatch "github.com/evanphx/json-patch"
)

// QuoteField is a custom type for quote fields (backed by string)
type QuoteField string

const (
	// Mark field
	Mark = QuoteField("MARK")
	// MarkChange field
	MarkChange = QuoteField("MARK_CHANGE")
	// MarkPercentChange field
	MarkPercentChange = QuoteField("MARK_PERCENT_CHANGE")
	// NetChange field
	NetChange = QuoteField("NET_CHANGE")
	// NetPercentChange field
	NetPercentChange = QuoteField("NET_CHANGE_PERCENT")
	// Bid field
	Bid = QuoteField("BID")
	// BidExchange field
	BidExchange = QuoteField("BID_EXCHANGE")
	// Ask field
	Ask = QuoteField("ASK")
	// AskExchange field
	AskExchange = QuoteField("ASK_EXCHANGE")
	// BidSize field
	BidSize = QuoteField("BID_SIZE")
	// AskSize field
	AskSize = QuoteField("ASK_SIZE")
	// Volume field
	Volume = QuoteField("VOLUME")
	// Open field
	Open = QuoteField("OPEN")
	// High52 field
	High52 = QuoteField("HIGH52")
	// Low52 field
	Low52 = QuoteField("LOW52")
	// High field
	High = QuoteField("HIGH")
	// Low field
	Low = QuoteField("LOW")
	// VWAP field
	VWAP = QuoteField("VWAP")
	// VolIdx field
	VolIdx = QuoteField("VOLATILITY_INDEX")
	// ImplVol field
	ImplVol = QuoteField("IMPLIED_VOLATILITY")
	// MMMove field
	MMMove = QuoteField("MARKET_MAKER_MOVE")
	// PercentIV field
	PercentIV = QuoteField("PERCENT_IV")
	// HistoricalVol30 field
	HistoricalVol30 = QuoteField("HISTORICAL_VOLATILITY_30_DAYS")
	// MktCap field
	MktCap = QuoteField("MARKET_CAP")
	// Beta field
	Beta = QuoteField("BETA")
	// PE field
	PE = QuoteField("PE")
	// InitMargin field
	InitMargin = QuoteField("INITIAL_MARGIN")
	// Last field
	Last = QuoteField("LAST")
	// LastSize field
	LastSize = QuoteField("LAST_SIZE")
	// LastExchange field
	LastExchange = QuoteField("LAST_EXCHANGE")
	// Rho field
	Rho = QuoteField("RHO")
	// BorrowStatus field
	BorrowStatus = QuoteField("BORROW_STATUS")
	// Delta field
	Delta = QuoteField("DELTA")
	// Gamma field
	Gamma = QuoteField("GAMMA")
	// Theta field
	Theta = QuoteField("THETA")
	// Vega field
	Vega = QuoteField("VEGA")
	// DivAmount field
	DivAmount = QuoteField("DIV_AMOUNT")
	// EPS field
	EPS = QuoteField("EPS")
	// ExdDivDate field
	ExdDivDate = QuoteField("EXD_DIV_DATE")
	// Yield field
	Yield = QuoteField("YIELD")
	// FrontVol field
	FrontVol = QuoteField("FRONT_VOLATILITY")
	// BackVol field
	BackVol = QuoteField("BACK_VOLATILITY")
	// VolDiff field
	VolDiff = QuoteField("VOLATILITY_DIFFERENCE")
	// Close field
	Close = QuoteField("CLOSE")
)

// QuoteRequestSignature is the parameter for a quote request
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

// QuoteStoredCache is the response from a quote request
type QuoteStoredCache struct {
	Items     []QuoteItem `json:"items"`
	Service   string      `json:"service"`
	RequestID string      `json:"requestId"`
	Ver       int         `json:"ver"`
}

// QuoteItem is a single item in a QuoteResponse
type QuoteItem struct {
	Symbol string      `json:"symbol"`
	Values quoteValues `json:"values"`
}

type quoteValues struct {
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
	rVerStr := gab.Search("header", "ver").String()
	rVer, _ := strconv.Atoi(rVerStr)
	rService = rService[1 : len(rService)-1]

	newState := s.CurrentState

	// get the patches
	patches := gab.S("body", "patches").Children()
	for _, patch := range patches {

		// fmt.Printf("%d - %s (%s)\n", rVer, patch.S("path").String(), patch.S("value").String())

		// TODO: implement actual error handling, currently using log.Fatal()
		// which is bad

		if patch.S("path").String() == "/error" {
			continue
		}

		path := patch.S("path").String()
		if path == "\"\"" {
			s.CurrentState.Quote = QuoteStoredCache{}
		}
		patch.Set(fmt.Sprintf("/quote%s", path[1:len(path)-1]), "path")

		bytesJSON, _ := patch.MarshalJSON()
		patchStr := "[" + string(bytesJSON) + "]"
		jspatch, err := jsonpatch.DecodePatch([]byte(patchStr))

		if err != nil {
			continue
		}

		byteState, _ := json.Marshal(newState)

		byteState, err = jspatch.Apply(byteState)
		if err != nil {
			continue
		}

		json.Unmarshal(byteState, &newState)
		newState.Quote.RequestID = rID
		newState.Quote.Service = rService
		newState.Quote.Ver = int(rVer)

	}

	s.CurrentState = newState
	s.NotificationChannel <- true
	s.TransactionChannel <- newState
}

// RequestQuote returns the quote for the relevant spec with the fields requested
func (s *Session) RequestQuote(specs QuoteRequestSignature) (*QuoteStoredCache, error) {

	// force capitalization of tickers, since the socket is case sensitive
	specs.Ticker = strings.ToUpper(specs.Ticker)
	uniqueID := fmt.Sprintf("%s-%d", specs.shortName(), s.QuoteRequestVers[specs.shortName()])

	if len(s.CurrentState.Quote.Items) != 0 &&
		s.CurrentState.Quote.Ver == s.specHash(fmt.Sprintf("%s-%d", specs.shortName(), s.QuoteRequestVers[specs.shortName()]-1)) {
		return &s.CurrentState.Quote, nil
	}

	hash := s.specHash(uniqueID)
	payload := gatewayRequestLoad{
		Payload: []gatewayRequest{
			{
				Header: gatewayHeader{
					Service: "quotes",
					ID:      "fluxQuotes",
					Ver:     hash,
				},
				Params: gatewayParams{
					Account:     "COMBINED ACCOUNT",
					Symbols:     strings.Split(specs.Ticker, ","),
					RefreshRate: specs.RefreshRate,
					QuoteFields: specs.Fields,
				},
			},
		},
	}

	s.QuoteMu.Lock()
	defer s.QuoteMu.Unlock()
	s.QuoteRequestVers[specs.shortName()]++
	s.sendJSON(payload)

	ctx, ctxCancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer ctxCancel()

	for {
		select {
		case <-ctx.Done():
			return nil, ErrNotReceivedInTime

		case <-s.NotificationChannel:
			if s.CurrentState.Quote.Ver == hash {
				cl := time.Now()
				for {
					if (len(s.CurrentState.Quote.Items) != 0 && s.CurrentState.Quote.Items[0].Values != quoteValues{}) {
						return &s.CurrentState.Quote, nil
					} else if time.Now().Sub(cl).Milliseconds() > 1000 {
						return nil, ErrNotReceivedInTime
					}
				}
			}
		}
	}
}
