/*
	Package configwatch watches local or remote config files for changes.
*/
package configwatch

import (
	"io"
	"time"

	"github.com/Symantec/Dominator/lib/log"
)

type Decoder func(reader io.Reader) (interface{}, error)

// Watch is designed to monitor configuration changes. Watch will monitor the
// provided URL for new data, calling the decoder and will send the decoded data
// to the channel that is returned. Decoded data are not sent if the checksum of
// the raw data has not changed since the last decoded data were sent to the
// channel. The URL is checked for changed data at least every checkInterval
// (for HTTP/HTTPS URLs) but may be checked more frequently (for local files).
func Watch(url string, checkInterval time.Duration,
	decoder Decoder, logger log.DebugLogger) (<-chan interface{}, error) {
	return watch(url, checkInterval, decoder, logger)
}
