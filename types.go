package flux

import (
	"github.com/adityaxdiwakar/tda-go"
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

type Heartbeat struct {
	Heartbeat int64 `json:"heartbeat"`
}
