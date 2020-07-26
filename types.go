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

type gatewayRequest struct {
	Service           string   `json:"service,omitempty"`
	ID                string   `json:"id,omitempty"`
	Ver               int      `json:"ver,omitempty"`
	Domain            string   `json:"domain,omitempty"`
	Platform          string   `json:"platform,omitempty"`
	Limit             int      `json:"limit,omitempty"`
	Pattern           string   `json:"pattern,omitempty"`
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

type heartbeat struct {
	Heartbeat int64 `json:"heartbeat"`
}
