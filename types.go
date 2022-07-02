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

type ChartRouteTableT map[string](chan RespTuple[ChartData])

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

type RespTuple[T any] struct {
	Body T
	Err  error
}

type ErroredBody struct {
	Message string `json:"message"`
}

type Body[T any] struct {
	Patches []PatchesBody[T] `json:"patches"`
}

type PatchesBody[T any] struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value T      `json:"value"`
}

type gatewayParams struct {
	Limit             int         `json:"limit,omitempty"`
	RefreshRate       int         `json:"refreshRate,omitempty"`
	ExtendedHours     bool        `json:"extendedHours,omitempty"`
	Domain            string      `json:"domain,omitempty"`
	Platform          string      `json:"platform,omitempty"`
	Pattern           string      `json:"pattern,omitempty"`
	Underlying        string      `json:"underlying,omitempty"`
	Exchange          string      `json:"exchange,omitempty"`
	UnderlyingSymbol  string      `json:"underlyingSymbol,omitempty"`
	Token             string      `json:"token,omitempty"`
	AccessToken       string      `json:"accessToken,omitempty"`
	Tag               string      `json:"tag,omitempty"`
	Symbol            string      `json:"symbol,omitempty"`
	AggregationPeriod string      `json:"timeAggregation,omitempty"`
	Range             string      `json:"range,omitempty"`
	Account           string      `json:"account,omitempty"`
	Studies           []string    `json:"studies,omitempty"`
	Symbols           []string    `json:"symbols,omitempty"`
	Filter            interface{} `json:"filter,omitempty"`
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
