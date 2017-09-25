package builder

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
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
		bootstrapStreams:          masterConfiguration.BootstrapStreams,
		imageStreams:              imageStreamsConfiguration.Streams,
		imageStreamsToAutoRebuild: imageStreamsToAutoRebuild,
		currentBuildLogs:          make(map[string]*bytes.Buffer),
		lastBuildResults:          make(map[string]buildResultType),
		packagerTypes:             masterConfiguration.PackagerTypes,
		variables:                 variables,
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
