package hypervisors

func (s probeStatus) String() string {
	switch s {
	case probeStatusNotYetProbed:
		return "not yet probed"
	case probeStatusGood:
		return "good"
	case probeStatusNoSrpc:
		return "no SRPC"
	case probeStatusNoService:
		return "no service"
	case probeStatusBad:
		return "bad"
	default:
		return "UNKNOWN"
	}
}
