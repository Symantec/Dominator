package dominator

import (
	"github.com/Symantec/Dominator/proto/sub"
)

type ConfigureSubsRequest sub.Configuration

type ConfigureSubsResponse struct{}

type DisableUpdatesRequest struct {
	Reason string
}

type DisableUpdatesResponse struct{}

type EnableUpdatesRequest struct {
	Reason string
}

type EnableUpdatesResponse struct{}

type GetSubsConfigurationRequest struct{}

type GetSubsConfigurationResponse sub.Configuration
