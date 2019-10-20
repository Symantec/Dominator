package repowatch

import (
	"time"

	"github.com/Symantec/Dominator/lib/log"
)

func Watch(remoteURL, localDirectory string, checkInterval time.Duration,
	metricName string, logger log.DebugLogger) (<-chan string, error) {
	return watch(remoteURL, localDirectory, checkInterval, metricName, logger)
}
