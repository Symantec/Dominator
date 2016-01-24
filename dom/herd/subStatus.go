package herd

import (
	"fmt"
)

func (status subStatus) string() string {
	switch status {
	case statusUnknown:
		return "unknown"
	case statusConnecting:
		return "connecting"
	case statusDNSError:
		return "DNS error"
	case statusConnectTimeout:
		return "connect timeout"
	case statusFailedToConnect:
		return "connect failed"
	case statusWaitingToPoll:
		return "waiting to poll"
	case statusPolling:
		return "polling"
	case statusFailedToPoll:
		return "poll failed"
	case statusSubNotReady:
		return "sub not ready"
	case statusImageNotReady:
		return "image not ready"
	case statusFetching:
		return "fetching"
	case statusFailedToFetch:
		return "fetch failed"
	case statusComputingUpdate:
		return "computing update"
	case statusUpdating:
		return "updating"
	case statusFailedToUpdate:
		return "update failed"
	case statusWaitingForNextFullPoll:
		return "waiting for next full poll"
	case statusSynced:
		return "synced"
	default:
		panic(fmt.Sprintf("unknown status: %d", status))
	}
}
