package common

type Hash [64]byte

type StatusResponse struct {
	Success     bool
	ErrorString string
}
