package builder

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	buildclient "github.com/Symantec/Dominator/imagebuilder/client"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imaginator"
)

const errNoSourceImage = "no source image: "
const errTooOldSourceImage = "too old source image: "

type dualBuildLogger struct {
	buffer *bytes.Buffer
	writer io.Writer
}

func checkPermission(builder imageBuilder, request proto.BuildImageRequest,
	authInfo *srpc.AuthInformation) error {
	if authInfo == nil || authInfo.HaveMethodAccess {
		return nil
	}
	if request.ExpiresIn > time.Hour*24 {
		return errors.New("maximum expiration time is 1 day")
	}
	if builder, ok := builder.(*imageStreamType); ok {
		for _, group := range builder.BuilderGroups {
			if _, ok := authInfo.GroupList[group]; ok {
				return nil
			}
		}
	}
	return errors.New("no permission to build: " + request.StreamName)
}

func needSourceImage(err error) (bool, string) {
	errString := err.Error()
	if index := strings.Index(errString, errNoSourceImage); index >= 0 {
		return true, errString[index+len(errNoSourceImage):]
	}
	if index := strings.Index(errString, errTooOldSourceImage); index >= 0 {
		return true, errString[index+len(errTooOldSourceImage):]
	}
	return false, ""
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
	var sleepUntil time.Time
	for ; ; time.Sleep(time.Until(sleepUntil)) {
		sleepUntil = time.Now().Add(minInterval)
		client, err := srpc.DialHTTP("tcp", b.imageServerAddress, 0)
		if err != nil {
			b.logger.Printf("%s: %s\n", b.imageServerAddress, err)
			continue
		}
		for _, streamName := range b.listStreamsToAutoRebuild() {
			_, _, err := b.build(client, proto.BuildImageRequest{
				StreamName: streamName,
				ExpiresIn:  minInterval * 2,
			},
				nil, nil)
			if err != nil {
				b.logger.Printf("Error building image: %s: %s\n",
					streamName, err)
			}
		}
		client.Close()
	}
}

func (b *Builder) buildImage(request proto.BuildImageRequest,
	authInfo *srpc.AuthInformation,
	logWriter io.Writer) (*image.Image, string, error) {
	if request.ExpiresIn < time.Minute*15 {
		return nil, "", errors.New("minimum expiration time is 15 minutes")
	}
	client, err := srpc.DialHTTP("tcp", b.imageServerAddress, 0)
	if err != nil {
		return nil, "", err
	}
	defer client.Close()
	img, name, err := b.build(client, request, authInfo, logWriter)
	if request.ReturnImage {
		return img, "", err
	}
	return nil, name, err
}

func (b *Builder) build(client *srpc.Client, request proto.BuildImageRequest,
	authInfo *srpc.AuthInformation,
	logWriter io.Writer) (*image.Image, string, error) {
	startTime := time.Now()
	builder := b.getImageBuilderWithReload(request.StreamName)
	if builder == nil {
		return nil, "", errors.New("unknown stream: " + request.StreamName)
	}
	if err := checkPermission(builder, request, authInfo); err != nil {
		return nil, "", err
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
	img, name, err := b.buildWithLogger(builder, client, request, authInfo,
		startTime, buildLog)
	finishTime := time.Now()
	b.buildResultsLock.Lock()
	defer b.buildResultsLock.Unlock()
	delete(b.currentBuildLogs, request.StreamName)
	b.lastBuildResults[request.StreamName] = buildResultType{
		name, startTime, finishTime, buildLog.Bytes(), err}
	if err == nil {
		b.logger.Printf("Built image for stream: %s in %s\n",
			request.StreamName, format.Duration(finishTime.Sub(startTime)))
	}
	return img, name, err
}

func (b *Builder) buildSomewhere(builder imageBuilder, client *srpc.Client,
	request proto.BuildImageRequest, authInfo *srpc.AuthInformation,
	buildLog buildLogger) (*image.Image, error) {
	if b.slaveDriver == nil {
		if authInfo == nil {
			b.logger.Printf("Auto building image for stream: %s\n",
				request.StreamName)
		} else {
			b.logger.Printf("%s requested building image for stream: %s\n",
				authInfo.Username, request.StreamName)
		}
		img, err := builder.build(b, client, request, buildLog)
		if err != nil {
			fmt.Fprintf(buildLog, "Error building image: %s\n", err)
		}
		return img, err
	} else {
		return b.buildOnSlave(client, request, authInfo, buildLog)
	}
}

func (b *Builder) buildWithLogger(builder imageBuilder, client *srpc.Client,
	request proto.BuildImageRequest, authInfo *srpc.AuthInformation,
	startTime time.Time, buildLog buildLogger) (*image.Image, string, error) {
	img, err := b.buildSomewhere(builder, client, request, authInfo, buildLog)
	if err != nil {
		if needSource, sourceImage := needSourceImage(err); needSource {
			if request.DisableRecursiveBuild {
				return nil, "", err
			}
			// Try to build source image.
			expiresIn := time.Hour
			if request.ExpiresIn > 0 {
				expiresIn = request.ExpiresIn
			}
			sourceReq := proto.BuildImageRequest{
				StreamName:   sourceImage,
				ExpiresIn:    expiresIn,
				MaxSourceAge: request.MaxSourceAge,
				Variables:    request.Variables,
			}
			if _, _, e := b.build(client, sourceReq, nil, buildLog); e != nil {
				return nil, "", e
			}
			img, err = b.buildSomewhere(builder, client, request, authInfo,
				buildLog)
		}
	}
	if err != nil {
		return nil, "", err
	}
	if request.ReturnImage {
		return img, "", nil
	}
	uploadStartTime := time.Now()
	if name, err := addImage(client, request, img); err != nil {
		fmt.Fprintln(buildLog, err)
		return nil, "", err
	} else {
		finishTime := time.Now()
		fmt.Fprintf(buildLog,
			"Uploaded %s in %s, total build duration: %s\n",
			name, format.Duration(finishTime.Sub(uploadStartTime)),
			format.Duration(finishTime.Sub(startTime)))
		return img, name, nil
	}
}

func (b *Builder) buildOnSlave(client *srpc.Client,
	request proto.BuildImageRequest, authInfo *srpc.AuthInformation,
	buildLog buildLogger) (*image.Image, error) {
	request.DisableRecursiveBuild = true
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
	keepSlave := false
	defer func() {
		if keepSlave {
			slave.Release()
		} else {
			slave.Destroy()
		}
	}()
	if authInfo == nil {
		b.logger.Printf("Auto building image on %s for stream: %s\n",
			slave, request.StreamName)
		fmt.Fprintf(buildLog, "Auto building image on %s for stream: %s\n",
			slave, request.StreamName)
	} else {
		b.logger.Printf("%s requested building image on %s for stream: %s\n",
			authInfo.Username, slave, request.StreamName)
		fmt.Fprintf(buildLog,
			"%s requested building image on %s for stream: %s\n",
			authInfo.Username, slave, request.StreamName)
	}
	var reply proto.BuildImageResponse
	err = buildclient.BuildImage(slave.GetClient(), request, &reply, buildLog)
	if err != nil {
		if needSource, _ := needSourceImage(err); needSource {
			keepSlave = true
		}
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
