package flux

import "errors"

var (
	ErrWsAlreadyOpen              = errors.New("error: connection already open")
	ErrGatewayUnsuccessful        = errors.New("error: gateway could not be requested")
	ErrProtocolUnestablished      = errors.New("error: could not establish protocol")
	ErrAuthenticationUnsuccessful = errors.New("error: authentication unsuccessful")
	ErrNotReceivedInTime          = errors.New("error: took too long to respond, try again")
)
