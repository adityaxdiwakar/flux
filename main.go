package flux

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/adityaxdiwakar/tda-go"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/gorilla/websocket"
)

type Session struct {
	TdaSession         tda.Session
	wsConn             *websocket.Conn
	ConfigUrl          string
	CurrentState       []byte
	CurrentChartHash   string
	TransactionChannel chan CachedData
}

type protocolPacketData struct {
	Ver       string `json:"ver"`
	Fmt       string `json:"fmt"`
	Heartbeat string `json:"heartbeat"`
}

type protocolResponse struct {
	Session string `json:"session"`
	Build   string `json:"build"`
	Ver     string `json:"ver"`
}

type ChartRequestSignature struct {
	Ticker string
	Range  string
	Width  string
}

type gatewayConfigResponse struct {
	APIURL           string `json:"apiUrl"`
	APIKey           string `json:"apiKey"`
	MobileGatewayURL struct {
		Livetrading string `json:"livetrading"`
		Papermoney  string `json:"papermoney"`
	} `json:"mobileGatewayUrl"`
	AuthURL string `json:"authUrl"`
}

type gatewayRequest struct {
	Service           string   `json:"service,omitempty"`
	ID                string   `json:"id,omitempty"`
	Ver               int      `json:"ver,omitempty"`
	Domain            string   `json:"domain,omitempty"`
	Platform          string   `json:"platform,omitempty"`
	Token             string   `json:"token,omitempty"`
	AccessToken       string   `json:"accessToken,omitempty"`
	Tag               string   `json:"tag,omitempty"`
	Symbol            string   `json:"symbol,omitempty"`
	AggregationPeriod string   `json:"aggregationPeriod,omitempty"`
	Studies           []string `json:"studies,omitempty"`
	Range             string   `json:"range,omitempty"`
}

type gatewayRequestLoad struct {
	Payload []gatewayRequest `json:"payload"`
}

var ErrWsAlreadyOpen = errors.New("error: connection already open")
var ErrGatewayUnsuccessful = errors.New("error: gateway could not be requested")
var ErrProtocolUnestablished = errors.New("error: could not establish protocol")
var ErrAuthenticationUnsuccessful = errors.New("error: authentication unsuccessful")

func New(creds tda.Session) (*Session, error) {
	s := &Session{
		TdaSession: creds,
	}
	if _, err := s.TdaSession.GetAccessToken(); err != nil {
		return nil, err
	}

	s.ConfigUrl = "https://trade.thinkorswim.com/v1/api/config"
	s.CurrentState = []byte(`{"data": ""}`)

	return s, nil
}

func (p *protocolResponse) IsEmpty() bool {
	if p.Build == "" {
		return true
	} else if p.Session == "" {
		return true
	} else if p.Ver == "" {
		return true
	} else {
		return false
	}
}

func (s *Session) dataAsChartObject() (*CachedData, error) {
	data := KeyCachedData{}
	err := json.Unmarshal(s.CurrentState, &data)
	if err != nil {
		return nil, err
	}
	return &data.Data, nil
}

func (s *Session) Open() error {
	var err error

	// if connection is open, bail out here
	if s.wsConn != nil {
		return ErrWsAlreadyOpen
	}

	// get the gateway url from the configuration endpoint
	gateway, err := s.Gateway()
	if err != nil {
		return err
	}

	s.wsConn, _, err = websocket.DefaultDialer.Dial(gateway, http.Header{})
	if err != nil {
		s.wsConn = nil
		return err
	}

	s.wsConn.SetCloseHandler(func(code int, text string) error {
		return nil
	})

	defer func() {
		if err != nil {
			s.wsConn.Close()
			s.wsConn = nil
		}
	}()

	establishProtocolPacket := protocolPacketData{
		Ver:       "25.*.*",
		Fmt:       "json-patches",
		Heartbeat: "5s",
	}

	err = s.wsConn.WriteJSON(establishProtocolPacket)
	if err != nil {
		return ErrProtocolUnestablished
	}

	var establishedProtocolResponse protocolResponse
	err = s.wsConn.ReadJSON(&establishedProtocolResponse)
	if err != nil {
		return ErrProtocolUnestablished
	}

	if establishedProtocolResponse.IsEmpty() {
		return ErrProtocolUnestablished
	}

	accessToken, err := s.TdaSession.GetAccessToken()
	if err != nil {
		return err
	}

	auth := gatewayRequest{
		Service:     "login",
		ID:          "login",
		Ver:         0,
		Domain:      "TOS",
		Platform:    "PROD",
		AccessToken: accessToken,
		Tag:         "TOSWeb",
	}

	request := gatewayRequestLoad{
		[]gatewayRequest{auth},
	}

	// push the authentication into the stream
	err = s.wsConn.WriteJSON(request)
	if err != nil {
		return ErrAuthenticationUnsuccessful
	}

	go s.listen()

	return nil
}

func (s *Session) Close() error {
	if s.wsConn != nil {
		err := s.wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1000, ""))
		if err != nil {
			return err
		}

		time.Sleep(time.Second)

		err = s.wsConn.Close()
		if err != nil {
			return err
		}

		s.wsConn = nil
	}

	return nil
}

type Heartbeat struct {
	Heartbeat int64 `json:"heartbeat"`
}

func (s *Session) listen() {
	for {
		_, message, err := s.wsConn.ReadMessage()
		if err != nil {
			log.Printf("error: closing websocket listen due to %v", err)
			break
		}

		if strings.Contains(string(message), "heartbeat") {
			continue
		}

		parsedJson, err := gabs.ParseJSON(message)
		serviceType := parsedJson.Search("payloadPatches", "0", "service")

		switch serviceType.String() {

		case `"login":`:
			log.Println("Successfully logged in")

		case `"chart"`:
			s.chartHandler(message, parsedJson)
		}
	}
}

type KeyCachedData struct {
	Data CachedData `json:"data"`
}

type CachedData struct {
	Symbol     string `json:"symbol"`
	Instrument struct {
		Symbol                 string `json:"symbol"`
		RootSymbol             string `json:"rootSymbol"`
		DisplaySymbol          string `json:"displaySymbol"`
		RootDisplaySymbol      string `json:"rootDisplaySymbol"`
		FutureOption           bool   `json:"futureOption"`
		Description            string `json:"description"`
		Multiplier             int    `json:"multiplier"`
		SpreadsSupported       bool   `json:"spreadsSupported"`
		Tradeable              bool   `json:"tradeable"`
		InstrumentType         string `json:"instrumentType"`
		ID                     int    `json:"id"`
		SourceType             string `json:"sourceType"`
		IsFutureProduct        bool   `json:"isFutureProduct"`
		HasOptions             bool   `json:"hasOptions"`
		Composite              bool   `json:"composite"`
		FractionalType         string `json:"fractionalType"`
		DaysToExpiration       int    `json:"daysToExpiration"`
		SpreadDaysToExpiration string `json:"spreadDaysToExpiration"`
		Cusip                  string `json:"cusip"`
		Industry               int    `json:"industry"`
		Spc                    int    `json:"spc"`
		ExtoEnabled            bool   `json:"extoEnabled"`
		Flags                  int    `json:"flags"`
	} `json:"instrument"`
	Timestamps []int64   `json:"timestamps"`
	Open       []float64 `json:"open"`
	High       []float64 `json:"high"`
	Low        []float64 `json:"low"`
	Close      []float64 `json:"close"`
	Volume     []int     `json:"volume"`
	Events     []struct {
		Symbol      string `json:"symbol"`
		CompanyName string `json:"companyName"`
		Website     string `json:"website"`
		IsActual    bool   `json:"isActual"`
		Time        int64  `json:"time"`
	} `json:"events"`
	Service    string `json:"service"`
	RequestID  string `json:"requestId"`
	RequestVer int    `json:"requestVer"`
}

type newChartObject struct {
	Op    string     `json:"op"`
	Path  string     `json:"path"`
	Value CachedData `json:"value"`
}

type updateChartObject struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value int    `json:"value"`
}

func (s *Session) chartHandler(msg []byte, gab *gabs.Container) {
	for _, patch := range gab.S("payloadPatches", "0", "patches").Children() {

		var err error
		bytesJson, err := patch.MarshalJSON()
		if err != nil {
			log.Fatal(err)
		}

		var modifiedChart []byte

		if patch.S("path").String() == "/error" {
			return
		}

		if patch.S("path").String() == `""` {
			newChart := newChartObject{}
			json.Unmarshal(bytesJson, &newChart)
			newChart.Path = "/data"
			modifiedChart, err = json.Marshal([]newChartObject{newChart})
		} else {
			updatedChart := updateChartObject{}
			json.Unmarshal(bytesJson, &updatedChart)
			updatedChart.Path = "/data" + updatedChart.Path
			modifiedChart, err = json.Marshal([]updateChartObject{updatedChart})
		}

		if err != nil {
			log.Fatal(err)
		}
		jspatch, err := jsonpatch.DecodePatch(modifiedChart)
		if err != nil {
			log.Fatal(err)
		}

		s.CurrentState, err = jspatch.Apply(s.CurrentState)
		if err != nil {
			log.Fatal(err)
		}

		if patch.S("path").String() == `""` {
			d, _ := s.dataAsChartObject()
			s.TransactionChannel <- *d
		}
	}
}

// Gateway returns the gateway URL as a string for the live trading connection
// to be made
func (s *Session) Gateway() (string, error) {
	res, err := s.TdaSession.HttpClient.Get("https://trade.thinkorswim.com/v1/api/config")
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", ErrGatewayUnsuccessful
	}

	defer res.Body.Close()

	var gatewayResponse gatewayConfigResponse

	json.NewDecoder(res.Body).Decode(&gatewayResponse)
	return gatewayResponse.MobileGatewayURL.Livetrading, nil
}

func (s *Session) RequestChart(specs ChartRequestSignature) (*CachedData, error) {
	// prepare request for gateway transaction

	s.TransactionChannel = make(chan CachedData)

	if s.CurrentChartHash == specs.shortName() {
		d, err := s.dataAsChartObject()
		if err != nil {
			return nil, err
		}
		return d, nil
	}

	s.CurrentChartHash = specs.shortName()

	req := gatewayRequest{
		Service:           "chart",
		ID:                "chart",
		Ver:               0,
		Symbol:            specs.Ticker,
		AggregationPeriod: specs.Width,
		Studies:           []string{},
		Range:             specs.Range,
	}
	payload := gatewayRequestLoad{[]gatewayRequest{req}}
	s.wsConn.WriteJSON(payload)

	internalChannel := make(chan CachedData)
	ctx, _ := context.WithTimeout(context.Background(), 750*time.Millisecond)

	go func() {
		for {
			select {

			case recvPayload := <-s.TransactionChannel:
				if strings.ToUpper(recvPayload.Symbol) == strings.ToUpper(specs.Ticker) {
					internalChannel <- recvPayload
				}

			case <-ctx.Done():
				break
			}
		}
	}()

	select {

	case recvPayload := <-internalChannel:
		return &recvPayload, nil

	case <-ctx.Done():
		return nil, errors.New("error: took too long to respond, try again")

	}

	//unreachable code
	return nil, nil
}

func (c *ChartRequestSignature) shortName() string {
	return fmt.Sprintf("%s@%s:%s", c.Ticker, c.Range, c.Width)
}
