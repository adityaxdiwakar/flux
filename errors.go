package flux

import "errors"

var (
	// ErrWsAlreadyOpen is returned if the connection being opened is already opened
	ErrWsAlreadyOpen = errors.New("error: connection already open")

	// ErrGatewayUnsuccessful is returned if the gateway helper could not retrieve the gateway URL
	ErrGatewayUnsuccessful = errors.New("error: gateway could not be requested")

	// ErrProtocolUnestablished is an error for the initial socket handshake
	// based on whether or not the proper session is returned
	ErrProtocolUnestablished = errors.New("error: could not establish protocol")

	// ErrAuthenticationUnsuccessful is returned if there is an authentication
	// error in the socket handshake and login command
	ErrAuthenticationUnsuccessful = errors.New("error: authentication unsuccessful")

	// ErrNotReceivedInTime is returned if the data being loaded could not be
	// found in the time enforcement
	ErrNotReceivedInTime = errors.New("error: took too long to respond, try again")
)
