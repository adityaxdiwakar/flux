package flux

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/adityaxdiwakar/tda-go"
	"github.com/gorilla/websocket"
)

// New takes the input of a tda.Session (see github.com/adityaxdiwakar/tda-go)
// and returns a flux Session which is used for all essentially library uses
func New(creds tda.Session) (*Session, error) {
	s := &Session{
		TdaSession: creds,
	}
	if _, err := s.TdaSession.GetAccessToken(); err != nil {
		return nil, err
	}

	s.ConfigUrl = "https://trade.thinkorswim.com/v1/api/config"
	s.CurrentState = []byte(`{"data": ""}`)
	s.TransactionChannel = make(chan cachedData)
	s.RequestVers = make(map[string]int)

	return s, nil
}

// Open is a method that opens the websocket connection with the TDAmeritrade
// server and returns an error if it is present
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

	// dial up the websocket dialer
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

	// initial message to be sent to receive a protocol message from the server
	establishProtocolPacket := protocolPacketData{
		Ver:       "25.*.*",
		Fmt:       "json-patches",
		Heartbeat: "5s",
	}

	err = s.wsConn.WriteJSON(establishProtocolPacket)
	if err != nil {
		return ErrProtocolUnestablished
	}

	// read the response from the server and parse it as a protocol response,
	// error if all fields are empty
	var establishedProtocolResponse protocolResponse
	err = s.wsConn.ReadJSON(&establishedProtocolResponse)
	if err != nil {
		return ErrProtocolUnestablished
	}

	if establishedProtocolResponse.IsEmpty() {
		return ErrProtocolUnestablished
	}

	// use github.com/adityaxdiwkar/tda-go to retrieve access token using
	// session credentials
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

	// launch goroutine to handle and listen the stream
	go s.listen()

	return nil
}

// Close sends a websocket.CloseMessage to the server and waits for closure
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
		// TODO: handle this better rather than ignoring the message
		if err != nil {
			continue
		}

		serviceType := parsedJson.Search("payloadPatches", "0", "service")

		switch serviceType.String() {

		case `"login":`:
			log.Println("Successfully logged in")

		case `"chart"`:
			s.chartHandler(message, parsedJson)
		}
	}
}
