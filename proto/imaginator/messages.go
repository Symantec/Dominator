package imaginator

import (
	"time"

	"github.com/Symantec/Dominator/lib/image"
)

type BuildImageRequest struct {
	StreamName     string
	ExpiresIn      time.Duration
	GitBranch      string
	MaxSourceAge   time.Duration
	ReturnImage    bool
	StreamBuildLog bool
	Variables      map[string]string
}

type BuildImageResponse struct {
	Image       *image.Image
	ImageName   string
	BuildLog    []byte
	ErrorString string
}
