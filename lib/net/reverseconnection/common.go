package reverseconnection

import (
	"time"
)

const (
	connectString = "200 Connected to ReverseDialer"
	urlPath       = "/_ReverseDialer_/connect"
)

type reverseDialerMessage struct {
	MinimumInterval time.Duration `json:",omitempty"`
	MaximumInterval time.Duration `json:",omitempty"`
}
