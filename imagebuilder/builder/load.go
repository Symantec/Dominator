package builder

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Symantec/Dominator/imageserver/client"
	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/url/urlutil"
)

func load(confUrl, variablesFile, stateDir, imageServerAddress string,
	imageRebuildInterval time.Duration, logger log.Logger) (
	*Builder, error) {
	masterConfiguration, err := masterConfiguration(confUrl)
	if err != nil {
		return nil, err
	}
	imageStreamsConfiguration, err := loadImageStreams(
		masterConfiguration.ImageStreamsUrl)
	if err != nil {
		return nil, err
	}
	imageStreamsToAutoRebuild := make([]string, 0)
	for name := range masterConfiguration.BootstrapStreams {
		imageStreamsToAutoRebuild = append(imageStreamsToAutoRebuild, name)
	}
	sort.Strings(imageStreamsToAutoRebuild)
	for _, name := range masterConfiguration.ImageStreamsToAutoRebuild {
		if _, ok := imageStreamsConfiguration.Streams[name]; ok {
			imageStreamsToAutoRebuild = append(imageStreamsToAutoRebuild, name)
		}
	}
	var variables map[string]string
	if variablesFile != "" {
		if err := libjson.ReadFromFile(variablesFile, &variables); err != nil {
			return nil, err
		}
	}
	if variables == nil {
		variables = make(map[string]string)
	}
	b := &Builder{
		stateDir:                  stateDir,
		imageServerAddress:        imageServerAddress,
		logger:                    logger,
		imageStreamsUrl:           masterConfiguration.ImageStreamsUrl,
		bootstrapStreams:          masterConfiguration.BootstrapStreams,
		imageStreams:              imageStreamsConfiguration.Streams,
		imageStreamsToAutoRebuild: imageStreamsToAutoRebuild,
		currentBuildLogs:          make(map[string]*bytes.Buffer),
		lastBuildResults:          make(map[string]buildResultType),
		packagerTypes:             masterConfiguration.PackagerTypes,
		variables:                 variables,
	}
	if err := b.makeRequiredDirectories(); err != nil {
		return nil, err
	}
	go b.rebuildImages(imageRebuildInterval)
	return b, nil
}

func masterConfiguration(url string) (*masterConfigurationType, error) {
	file, err := urlutil.Open(url)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var configuration masterConfigurationType
	decoder := json.NewDecoder(bufio.NewReader(file))
	if err := decoder.Decode(&configuration); err != nil {
		return nil, fmt.Errorf("error reading configuration from: %s: %s",
			url, err)
	}
	for _, stream := range configuration.BootstrapStreams {
		if _, ok := configuration.PackagerTypes[stream.PackagerType]; !ok {
			return nil, fmt.Errorf("packager type: \"%s\" unknown",
				stream.PackagerType)
		}
		if stream.Filter != nil {
			if err := stream.Filter.Compile(); err != nil {
				return nil, err
			}
		}
	}
	return &configuration, nil
}

func loadImageStreams(url string) (*imageStreamsConfigurationType, error) {
	if url == "" {
		return &imageStreamsConfigurationType{}, nil
	}
	file, err := urlutil.Open(url)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var configuration imageStreamsConfigurationType
	decoder := json.NewDecoder(bufio.NewReader(file))
	if err := decoder.Decode(&configuration); err != nil {
		return nil, fmt.Errorf("error reading image streams from: %s: %s",
			url, err)
	}
	return &configuration, nil
}

func (b *Builder) makeRequiredDirectories() error {
	imageServer, err := srpc.DialHTTP("tcp", b.imageServerAddress, 0)
	if err != nil {
		b.logger.Println(err)
		return nil
	}
	defer imageServer.Close()
	directoryList, err := client.ListDirectories(imageServer)
	if err != nil {
		b.logger.Println(err)
		return nil
	}
	directories := make(map[string]struct{}, len(directoryList))
	for _, directory := range directoryList {
		directories[directory.Name] = struct{}{}
	}
	streamNames := b.listAllStreamNames()
	for _, streamName := range streamNames {
		if _, ok := directories[streamName]; ok {
			continue
		}
		pathComponents := strings.Split(streamName, "/")
		for index := range pathComponents {
			partPath := strings.Join(pathComponents[0:index+1], "/")
			if _, ok := directories[partPath]; ok {
				continue
			}
			if err := client.MakeDirectory(imageServer, partPath); err != nil {
				return err
			}
			b.logger.Printf("Created missing directory: %s\n", partPath)
			directories[partPath] = struct{}{}
		}
	}
	return nil
}

func (b *Builder) reloadNormalStreamsConfiguration() error {
	imageStreamsConfiguration, err := loadImageStreams(b.imageStreamsUrl)
	if err != nil {
		return err
	}
	b.logger.Println("Reloaded streams streams configuration")
	b.streamsLock.Lock()
	b.imageStreams = imageStreamsConfiguration.Streams
	b.streamsLock.Unlock()
	if err := b.makeRequiredDirectories(); err != nil {
		return err
	}
	return nil
}
