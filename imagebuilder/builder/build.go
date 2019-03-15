package builder

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	buildclient "github.com/Symantec/Dominator/imagebuilder/client"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imaginator"
)

type dualBuildLogger struct {
	buffer *bytes.Buffer
	writer io.Writer
}

func (bl *dualBuildLogger) Bytes() []byte {
	return bl.buffer.Bytes()
}

func (bl *dualBuildLogger) Write(p []byte) (int, error) {
	return bl.writer.Write(p)
}

func (b *Builder) rebuildImages(minInterval time.Duration) {
	if minInterval < 1 {
		return
	}
	firstTime := true // HACK.
	var sleepUntil time.Time
	for ; ; time.Sleep(time.Until(sleepUntil)) {
		sleepUntil = time.Now().Add(minInterval)
		if firstTime {
			firstTime = false
			continue
		}
		client, err := srpc.DialHTTP("tcp", b.imageServerAddress, 0)
		if err != nil {
			b.logger.Println(err)
			continue
		}
		for _, streamName := range b.listStreamsToAutoRebuild() {
			_, _, err := b.build(client, proto.BuildImageRequest{
				StreamName: streamName,
				ExpiresIn:  minInterval * 2,
			},
				nil)
			if err != nil {
				b.logger.Printf("Error building image: %s: %s\n",
					streamName, err)
			}
		}
		client.Close()
	}
}

func (b *Builder) buildImage(request proto.BuildImageRequest,
	logWriter io.Writer) (*image.Image, string, error) {
	client, err := srpc.DialHTTP("tcp", b.imageServerAddress, 0)
	if err != nil {
		return nil, "", err
	}
	defer client.Close()
	img, name, err := b.build(client, request, logWriter)
	return img, name, err
}

func (b *Builder) build(client *srpc.Client, request proto.BuildImageRequest,
	logWriter io.Writer) (*image.Image, string, error) {
	startTime := time.Now()
	builder := b.getImageBuilderWithReload(request.StreamName)
	if builder == nil {
		return nil, "", errors.New("unknown stream: " + request.StreamName)
	}
	buildLogBuffer := &bytes.Buffer{}
	b.buildResultsLock.Lock()
	b.currentBuildLogs[request.StreamName] = buildLogBuffer
	b.buildResultsLock.Unlock()
	var buildLog buildLogger
	if logWriter == nil {
		buildLog = buildLogBuffer
	} else {
		buildLog = &dualBuildLogger{
			buffer: buildLogBuffer,
			writer: io.MultiWriter(buildLogBuffer, logWriter),
		}
	}
	var img *image.Image
	var err error
	if b.slaveDriver == nil {
		b.logger.Printf("Building new image for stream: %s\n",
			request.StreamName)
		img, err = builder.build(b, client, request, buildLog)
	} else {
		img, err = b.buildOnSlave(client, request, buildLog)
	}
	finishTime := time.Now()
	var name string
	if err != nil {
		fmt.Fprintf(buildLog, "Error building image: %s\n", err)
	} else {
		if !request.ReturnImage {
			uploadStartTime := time.Now()
			name, err = addImage(client, request, img)
			finishTime = time.Now()
			if err != nil {
				fmt.Fprintln(buildLog, err)
			} else {
				fmt.Fprintf(buildLog,
					"Uploaded %s in %s, total build duration: %s\n",
					name, format.Duration(finishTime.Sub(uploadStartTime)),
					format.Duration(finishTime.Sub(startTime)))
			}
		}
		b.logger.Printf("Built image for stream: %s in %s\n",
			request.StreamName, format.Duration(finishTime.Sub(startTime)))
	}
	b.buildResultsLock.Lock()
	defer b.buildResultsLock.Unlock()
	delete(b.currentBuildLogs, request.StreamName)
	b.lastBuildResults[request.StreamName] = buildResultType{
		name, startTime, finishTime, buildLog.Bytes(), err}
	return img, name, nil
}

func (b *Builder) buildOnSlave(client *srpc.Client,
	request proto.BuildImageRequest,
	buildLog buildLogger) (*image.Image, error) {
	request.ReturnImage = true
	request.StreamBuildLog = true
	if len(request.Variables) < 1 {
		request.Variables = b.variables
	} else if len(b.variables) > 0 {
		variables := make(map[string]string,
			len(b.variables)+len(request.Variables))
		for key, value := range b.variables {
			variables[key] = value
		}
		for key, value := range request.Variables {
			variables[key] = value
		}
		request.Variables = variables
	}
	slave, err := b.slaveDriver.GetSlave()
	if err != nil {
		return nil, fmt.Errorf("error getting slave: %s", err)
	}
	defer slave.Destroy()
	b.logger.Printf("Building new image on %s for stream: %s\n",
		slave, request.StreamName)
	fmt.Fprintf(buildLog, "Building new image on %s for stream: %s\n",
		slave, request.StreamName)
	var reply proto.BuildImageResponse
	err = buildclient.BuildImage(slave.GetClient(), request, &reply, buildLog)
	if err != nil {
		return nil, err
	}
	return reply.Image, nil
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
