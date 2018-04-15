package configwatch

import (
	"io"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/url/urlutil"
)

func watch(url string, checkInterval time.Duration,
	decoder Decoder, logger log.DebugLogger) (<-chan interface{}, error) {
	rawChannel, err := urlutil.WatchUrl(url, checkInterval, logger)
	if err != nil {
		return nil, err
	}
	configChannel := make(chan interface{}, 1)
	go watchLoop(rawChannel, configChannel, decoder, logger)
	return configChannel, nil
}

func watchLoop(rawChannel <-chan io.ReadCloser,
	configChannel chan<- interface{}, decoder Decoder, logger log.DebugLogger) {
	for reader := range rawChannel {
		if config, err := decoder(reader); err != nil {
			logger.Println(err)
		} else {
			configChannel <- config
		}
		reader.Close()
	}
	close(configChannel)
}
