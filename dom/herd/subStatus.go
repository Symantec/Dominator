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
	case statusConnectionRefused:
		return "connection refused"
	case statusNoRouteToHost:
		return "no route to host"
	case statusConnectTimeout:
		return "connect timeout"
	case statusMissingCertificate:
		return "connect failed: missing certificate"
	case statusBadCertificate:
		return "connect failed: bad certificate"
	case statusFailedToConnect:
		return "connect failed"
	case statusWaitingToPoll:
		return "waiting to poll"
	case statusPolling:
		return "polling"
	case statusPollDenied:
		return "poll denied"
	case statusFailedToPoll:
		return "poll failed"
	case statusSubNotReady:
		return "sub not ready"
	case statusImageUndefined:
		return "image undefined"
	case statusImageNotReady:
		return "image not ready"
	case statusNotEnoughFreeSpace:
		return "insufficient space"
	case statusFetching:
		return "fetching"
	case statusFetchDenied:
		return "fetch denied"
	case statusFailedToFetch:
		return "fetch failed"
	case statusPushing:
		return "pushing"
	case statusPushDenied:
		return "push denied"
	case statusFailedToPush:
		return "failed to push"
	case statusFailedToGetObject:
		return "failed to get object"
	case statusComputingUpdate:
		return "computing update"
	case statusSendingUpdate:
		return "sending update"
	case statusMissingComputedFile:
		return "missing computed file"
	case statusUpdatesDisabled:
		return "updates disabled"
	case statusUnsafeUpdate:
		return "unsafe update"
	case statusUpdating:
		return "updating"
	case statusUpdateDenied:
		return "update denied"
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

func (status subStatus) html() string {
	switch status {
	case statusUnsafeUpdate:
		return `<font color="red">` + status.String() + "</font>"
	default:
		return status.String()
	}
}
