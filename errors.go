package flux

import "errors"

var ErrWsAlreadyOpen = errors.New("error: connection already open")
var ErrGatewayUnsuccessful = errors.New("error: gateway could not be requested")
var ErrProtocolUnestablished = errors.New("error: could not establish protocol")
var ErrAuthenticationUnsuccessful = errors.New("error: authentication unsuccessful")
