package configwatch

import (
	"io"
	"time"

	"github.com/Symantec/Dominator/lib/log"
)

type Decoder func(reader io.Reader) (interface{}, error)

func Watch(url string, checkInterval time.Duration,
	decoder Decoder, logger log.DebugLogger) (<-chan interface{}, error) {
	return watch(url, checkInterval, decoder, logger)
}
