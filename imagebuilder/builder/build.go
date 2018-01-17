package builder

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/srpc"
)

func (b *Builder) rebuildImages(minInterval time.Duration) {
	if minInterval < 1 {
		return
	}
	var sleepUntil time.Time
	for ; ; time.Sleep(time.Until(sleepUntil)) {
		sleepUntil = time.Now().Add(minInterval)
		client, err := srpc.DialHTTP("tcp", b.imageServerAddress, 0)
		if err != nil {
			b.logger.Println(err)
			continue
		}
		for _, streamName := range b.listStreamsToAutoRebuild() {
			_, _, err := b.build(client, streamName, minInterval*2, "", 0)
			if err != nil {
				b.logger.Printf("Error building image: %s: %s\n",
					streamName, err)
			}
		}
		client.Close()
	}
}

func (b *Builder) buildImage(streamName string,
	expiresIn time.Duration, gitBranch string, maxSourceAge time.Duration) (
	string, []byte, error) {
	client, err := srpc.DialHTTP("tcp", b.imageServerAddress, 0)
	if err != nil {
		return "", nil, err
	}
	defer client.Close()
	name, buildLog, err := b.build(client, streamName, expiresIn, gitBranch,
		maxSourceAge)
	log := make([]byte, len(buildLog))
	copy(log, buildLog)
	return name, log, err
}

func (b *Builder) build(client *srpc.Client, streamName string,
	expiresIn time.Duration, gitBranch string, maxSourceAge time.Duration) (
	string, []byte, error) {
	startTime := time.Now()
	builder := b.getImageBuilderWithReload(streamName)
	if builder == nil {
		return "", nil, errors.New("unknown stream: " + streamName)
	}
	b.logger.Printf("Building new image for stream: %s\n", streamName)
	buildLog := &bytes.Buffer{}
	b.buildResultsLock.Lock()
	b.currentBuildLogs[streamName] = buildLog
	b.buildResultsLock.Unlock()
	name, err := builder.build(b, client, streamName, expiresIn, gitBranch,
		maxSourceAge, buildLog)
	if err == nil {
		buildDuration := format.Duration(time.Since(startTime))
		fmt.Fprintf(buildLog, "Total build duration: %s\n", buildDuration)
		b.logger.Printf("Built image for stream: %s in %s\n",
			streamName, buildDuration)
	} else {
		fmt.Fprintf(buildLog, "Error building image: %s\n", err)
	}
	b.buildResultsLock.Lock()
	defer b.buildResultsLock.Unlock()
	delete(b.currentBuildLogs, streamName)
	b.lastBuildResults[streamName] = buildResultType{
		name, buildLog.Bytes(), err}
	return name, buildLog.Bytes(), err
}

func (b *Builder) getImageBuilder(streamName string) imageBuilder {
	if stream := b.getBootstrapStream(streamName); stream != nil {
		return stream
	}
	if stream := b.getNormalStream(streamName); stream != nil {
		return stream
	}
	// Ensure a nil interface is returned, not a stream with value == nil.
	return nil
}

func (b *Builder) getImageBuilderWithReload(streamName string) imageBuilder {
	if stream := b.getImageBuilder(streamName); stream != nil {
		return stream
	}
	if err := b.reloadNormalStreamsConfiguration(); err != nil {
		b.logger.Printf("Error reloading configuration: %s\n", err)
		return nil
	}
	return b.getImageBuilder(streamName)
}

func (b *Builder) getCurrentBuildLog(streamName string) ([]byte, error) {
	b.buildResultsLock.RLock()
	defer b.buildResultsLock.RUnlock()
	if result, ok := b.currentBuildLogs[streamName]; !ok {
		return nil, errors.New("unknown image: " + streamName)
	} else {
		log := make([]byte, result.Len())
		copy(log, result.Bytes())
		return log, nil
	}
}

func (b *Builder) getLatestBuildLog(streamName string) ([]byte, error) {
	b.buildResultsLock.RLock()
	defer b.buildResultsLock.RUnlock()
	if result, ok := b.lastBuildResults[streamName]; !ok {
		return nil, errors.New("unknown image: " + streamName)
	} else {
		log := make([]byte, len(result.buildLog))
		copy(log, result.buildLog)
		return log, nil
	}
}
