package flux

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

type gatewayConfigResponse struct {
	APIURL           string `json:"apiUrl"`
	APIKey           string `json:"apiKey"`
	MobileGatewayURL struct {
		Livetrading string `json:"livetrading"`
		Papermoney  string `json:"papermoney"`
	} `json:"mobileGatewayUrl"`
	AuthURL string `json:"authUrl"`
}

type gatewayHeader struct {
	Service string `json:"service,omitempty"`
	ID      string `json:"id,omitempty"`
	Ver     int    `json:"ver"`
}

type gatewayParams struct {
	Limit             int          `json:"limit,omitempty"`
	Domain            string       `json:"domain,omitempty"`
	Platform          string       `json:"platform,omitempty"`
	Pattern           string       `json:"pattern,omitempty"`
	Underlying        string       `json:"underlying,omitempty"`
	Exchange          string       `json:"exchange,omitempty"`
	UnderlyingSymbol  string       `json:"underlyingSymbol,omitempty"`
	Token             string       `json:"token,omitempty"`
	AccessToken       string       `json:"accessToken,omitempty"`
	Tag               string       `json:"tag,omitempty"`
	Symbol            string       `json:"symbol,omitempty"`
	AggregationPeriod string       `json:"timeAggregation,omitempty"`
	Range             string       `json:"range,omitempty"`
	Studies           []string     `json:"studies,omitempty"`
	Filter            interface{}  `json:"filter,omitempty"`
	Account           string       `json:"account,omitempty"`
	Symbols           []string     `json:"symbols,omitempty"`
	QuoteFields       []QuoteField `json:"fields,omitempty"`
	RefreshRate       int          `json:"refreshRate,omitempty"`
	ExtendedHours     bool         `json:"extendedHours,omitempty"`
}

type gatewayRequest struct {
	Header gatewayHeader `json:"header"`
	Params gatewayParams `json:"params"`
}

type gatewayRequestLoad struct {
	Payload []gatewayRequest `json:"payload"`
}

type heartbeat struct {
	Heartbeat int64 `json:"heartbeat"`
}
