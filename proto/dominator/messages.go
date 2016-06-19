package dominator

import (
	"github.com/Symantec/Dominator/proto/sub"
)

type DisableUpdatesRequest struct {
	Reason string
}

type DisableUpdatesResponse struct{}

type EnableUpdatesRequest struct {
	Reason string
}

type EnableUpdatesResponse struct{}

type ConfigureSubsRequest sub.Configuration

type ConfigureSubsResponse struct{}
