package hypervisors

func (s probeStatus) String() string {
	switch s {
	case probeStatusNotYetProbed:
		return "not yet probed"
	case probeStatusConnected:
		return "connected"
	case probeStatusNoSrpc:
		return "no SRPC"
	case probeStatusNoService:
		return "no service"
	case probeStatusConnectionRefused:
		return "connection refused"
	case probeStatusUnreachable:
		return "unreachable"
	case probeStatusOff:
		return "off"
	default:
		return "UNKNOWN"
	}
}
