package flux

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/adityaxdiwakar/tda-go"
	"github.com/gorilla/websocket"
)

// New takes the input of a tda.Session (see github.com/adityaxdiwakar/tda-go)
// and returns a flux Session which is used for all essentially library uses
func New(creds tda.Session, debug bool) (*Session, error) {
	s := &Session{
		TdaSession: creds,
	}
	if _, err := s.TdaSession.GetAccessToken(); err != nil {
		return nil, err
	}

	s.DebugFlag = debug
	s.ConfigURL = "https://trade.thinkorswim.com/v1/api/config"
	s.CurrentState = storedCache{}
	s.TransactionChannel = make(chan storedCache, 5)
	s.NotificationChannel = make(chan bool, 50)
	s.ChartRequestVers = make(map[string]int)
	s.QuoteRequestVers = make(map[string]int)
	s.OptionQuoteRequestVers = make(map[string]int)
	s.SearchRequestVers = make(map[string]int)
	s.OptionSeriesRequestVers = make(map[string]int)
	s.OptionChainGetRequestVers = make(map[string]int)
	s.Established = false

	return s, nil
}

// Reset resets the state; it does not reset the connection
func (s *Session) Reset() error {
	if _, err := s.TdaSession.GetAccessToken(); err != nil {
		return err
	}

	s.ConfigURL = "https://trade.thinkorswim.com/v1/api/config"
	s.CurrentState = storedCache{}
	s.TransactionChannel = make(chan storedCache, 5)
	s.NotificationChannel = make(chan bool, 50)
	s.ChartRequestVers = make(map[string]int)
	s.QuoteRequestVers = make(map[string]int)
	s.OptionQuoteRequestVers = make(map[string]int)
	s.SearchRequestVers = make(map[string]int)
	s.OptionSeriesRequestVers = make(map[string]int)
	s.OptionChainGetRequestVers = make(map[string]int)
	s.Established = false

	return nil
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
		Ver:       "26.*.*",
		Fmt:       "json-patches-structured",
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

	request := gatewayRequestLoad{
		Payload: []gatewayRequest{
			{
				Header: gatewayHeader{
					Service: "login",
					ID:      "login",
					Ver:     0,
				},
				Params: gatewayParams{
					Domain:      "TOS",
					Platform:    "PROD",
					AccessToken: accessToken,
					Tag:         "TOSWeb",
				},
			},
		},
	}

	// push the authentication into the stream
	err = s.wsConn.WriteJSON(request)
	if err != nil {
		return ErrAuthenticationUnsuccessful
	}

	// launch goroutine to handle and listen the stream
	go s.listen()
	go s.reconnectHandler()

	s.Established = true
	return nil
}

func (s *Session) sendJSON(v interface{}) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return s.wsConn.WriteJSON(v)
}

func (s *Session) reconnectHandler() {
	// latch to only establish a single handler (singleton)
	if s.HandlerWorking == true {
		return
	}
	s.HandlerWorking = true

	// at the end of this function, delete this handler
	defer func() {
		s.HandlerWorking = false
	}()

	// reconnect every 20 minutes
	for range time.Tick(20 * time.Minute) {
		s.MutexLock = true
		s.Mu.Lock()
		fmt.Printf("\n")
		log.Println("[FLUX] Restarting connection, attempting disconnect...")
		s.Close()
		log.Println("[FLUX] Attempting reconnect...")
		s.Open()
		log.Println("[FLUX] Connected")
		time.Sleep(250 * time.Millisecond)
		s.Mu.Unlock()
		s.MutexLock = false
	}
}

// Close sends a websocket.CloseMessage to the server and waits for closure
func (s *Session) Close() error {
	if s.wsConn != nil {
		s.Established = false
		s.wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1000, ""))

		time.Sleep(time.Second)

		err := s.wsConn.Close()
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
			if s.MutexLock == true {
				log.Printf("[FLUX] Disconnected with routine restart imminent...")
			} else {
				log.Printf("error: closing websocket listen due to %v", err)
				// trying to connect again
				s.Reset()
				s.Open()
			}
			break
		}

		if s.DebugFlag {
			fmt.Println(string(message))
		}

		if strings.Contains(string(message), "heartbeat") {
			continue
		}

		parsedJSON, err := gabs.ParseJSON(message)
		// TODO: handle this better rather than ignoring the message
		if err != nil {
			continue
		}

		for _, child := range parsedJSON.S("payload").Children() {

			serviceType := child.Search("header", "service")

			switch serviceType.String() {

			case `"login":`:
				log.Println("Successfully logged in")

			case `"chart_v27"`:
				// TODO: change this to chart_v27
				s.chartHandler(message, child)

			case `"instrument_search"`:
				s.searchHandler(message, child)

			case `"optionSeries"`:
				s.optionSeriesHandler(message, child)

			case `"option_chain/get"`:
				s.optionChainGetHandler(message, child)

			case `"quotes"`:
				go s.quoteHandler(message, child)

			case `"quotes/options"`:
				s.optionQuoteHandler(message, child)

			}

		}
	}
}
