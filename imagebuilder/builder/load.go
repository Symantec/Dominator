package builder

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/configwatch"
	libjson "github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/slavedriver"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/url/urlutil"
)

func imageStreamsDecoder(reader io.Reader) (interface{}, error) {
	var config imageStreamsConfigurationType
	decoder := json.NewDecoder(bufio.NewReader(reader))
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("error reading image streams: %s", err)
	}
	return &config, nil
}

func load(confUrl, variablesFile, stateDir, imageServerAddress string,
	imageRebuildInterval time.Duration, slaveDriver *slavedriver.SlaveDriver,
	logger log.DebugLogger) (*Builder, error) {
	err := syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, "")
	if err != nil {
		return nil, fmt.Errorf("error making mounts private: %s", err)
	}
	masterConfiguration, err := masterConfiguration(confUrl)
	if err != nil {
		return nil, fmt.Errorf("error getting master configuration: %s", err)
	}
	imageStreamsToAutoRebuild := make([]string, 0)
	for name := range masterConfiguration.BootstrapStreams {
		imageStreamsToAutoRebuild = append(imageStreamsToAutoRebuild, name)
	}
	sort.Strings(imageStreamsToAutoRebuild)
	for _, name := range masterConfiguration.ImageStreamsToAutoRebuild {
		imageStreamsToAutoRebuild = append(imageStreamsToAutoRebuild, name)
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
		bindMounts:                masterConfiguration.BindMounts,
		stateDir:                  stateDir,
		imageServerAddress:        imageServerAddress,
		logger:                    logger,
		imageStreamsUrl:           masterConfiguration.ImageStreamsUrl,
		bootstrapStreams:          masterConfiguration.BootstrapStreams,
		imageStreamsToAutoRebuild: imageStreamsToAutoRebuild,
		slaveDriver:               slaveDriver,
		currentBuildLogs:          make(map[string]*bytes.Buffer),
		lastBuildResults:          make(map[string]buildResultType),
		packagerTypes:             masterConfiguration.PackagerTypes,
		variables:                 variables,
	}
	for name, stream := range b.bootstrapStreams {
		stream.builder = b
		stream.name = name
	}
	imageStreamsConfigChannel, err := configwatch.WatchWithCache(
		masterConfiguration.ImageStreamsUrl,
		time.Second*time.Duration(
			masterConfiguration.ImageStreamsCheckInterval), imageStreamsDecoder,
		filepath.Join(stateDir, "image-streams.json"),
		time.Second*5, logger)
	if err != nil {
		return nil, err
	}
	go b.watchConfigLoop(imageStreamsConfigChannel)
	go b.rebuildImages(imageRebuildInterval)
	return b, nil
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
		return nil, fmt.Errorf("error decoding image streams from: %s: %s",
			url, err)
	}
	return &configuration, nil
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

func (b *Builder) delayMakeRequiredDirectories(abortNotifier <-chan struct{}) {
	timer := time.NewTimer(time.Second * 5)
	select {
	case <-abortNotifier:
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
		b.makeRequiredDirectories()
	}
}

func (b *Builder) makeRequiredDirectories() error {
	imageServer, err := srpc.DialHTTP("tcp", b.imageServerAddress, 0)
	if err != nil {
		b.logger.Printf("%s: %s\n", b.imageServerAddress, err)
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
	return b.updateImageStreams(imageStreamsConfiguration)
}

func (b *Builder) updateImageStreams(
	imageStreamsConfiguration *imageStreamsConfigurationType) error {
	for name, stream := range imageStreamsConfiguration.Streams {
		stream.builder = b
		stream.name = name
	}
	b.streamsLock.Lock()
	b.imageStreams = imageStreamsConfiguration.Streams
	b.streamsLock.Unlock()
	return b.makeRequiredDirectories()
}

func (b *Builder) watchConfigLoop(configChannel <-chan interface{}) {
	firstLoadNotifier := make(chan struct{})
	go b.delayMakeRequiredDirectories(firstLoadNotifier)
	for rawConfig := range configChannel {
		imageStreamsConfig, ok := rawConfig.(*imageStreamsConfigurationType)
		if !ok {
			b.logger.Printf("received unknown type over config channel")
			continue
		}
		if firstLoadNotifier != nil {
			firstLoadNotifier <- struct{}{}
			close(firstLoadNotifier)
			firstLoadNotifier = nil
		}
		b.logger.Println("received new image streams configuration")
		b.updateImageStreams(imageStreamsConfig)
	}
}
