package repowatch

import (
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func Watch(remoteURL, localDirectory string, checkInterval time.Duration,
	metricDirectory string, logger log.DebugLogger) (<-chan string, error) {
	return watch(remoteURL, localDirectory, checkInterval, metricDirectory,
		logger)
}
