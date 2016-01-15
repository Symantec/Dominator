package herd

import (
	"fmt"
)

func (status subStatus) string() string {
	switch {
	case status == statusUnknown:
		return "unknown"
	case status == statusConnecting:
		return "connecting"
	case status == statusDNSError:
		return "DNS error"
	case status == statusFailedToConnect:
		return "connect failed"
	case status == statusWaitingToPoll:
		return "waiting to poll"
	case status == statusPolling:
		return "polling"
	case status == statusFailedToPoll:
		return "poll failed"
	case status == statusSubNotReady:
		return "sub not ready"
	case status == statusImageNotReady:
		return "image not ready"
	case status == statusFetching:
		return "fetching"
	case status == statusFailedToFetch:
		return "fetch failed"
	case status == statusComputingUpdate:
		return "computing update"
	case status == statusUpdating:
		return "updating"
	case status == statusFailedToUpdate:
		return "update failed"
	case status == statusSynced:
		return "synced"
	default:
		panic(fmt.Sprintf("unknown status: %d", status))
	}
}
