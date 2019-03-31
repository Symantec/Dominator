package proxy

import "time"

type ConnectRequest struct {
	Address string
	Network string
	Timeout time.Duration
}

type ConnectResponse struct {
	Error string
}
