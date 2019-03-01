package builder

import (
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/triggers"
)

type buildLogger interface {
	Bytes() []byte
	io.Writer
}

type imageBuilder interface {
	build(b *Builder, client *srpc.Client, streamName string,
		expiresIn time.Duration, gitBranch string, maxSourceAge time.Duration,
		uploadImage bool, buildLog buildLogger) (*image.Image, string, error)
}

type bootstrapStream struct {
	builder          *Builder
	name             string
	BootstrapCommand []string
	*filter.Filter
	PackagerType string
}

type buildResultType struct {
	imageName  string
	startTime  time.Time
	finishTime time.Time
	buildLog   []byte
	error      error
}

type masterConfigurationType struct {
	BootstrapStreams          map[string]*bootstrapStream `json:",omitempty"`
	ImageStreamsCheckInterval uint                        `json:",omitempty"`
	ImageStreamsToAutoRebuild []string                    `json:",omitempty"`
	ImageStreamsUrl           string                      `json:",omitempty"`
	PackagerTypes             map[string]packagerType     `json:",omitempty"`
}

type manifestConfigType struct {
	SourceImage string
	*filter.Filter
}

type manifestType struct {
	filter          *filter.Filter
	sourceImageInfo *sourceImageInfoType
}

type imageStreamType struct {
	builder           *Builder
	name              string
	ManifestUrl       string
	ManifestDirectory string
}

type imageStreamsConfigurationType struct {
	Streams map[string]*imageStreamType `json:",omitempty"`
}

type argList []string

type listCommandType struct {
	ArgList        argList
	SizeMultiplier uint64
}

type packagerType struct {
	CleanCommand   argList
	InstallCommand argList
	ListCommand    listCommandType
	RemoveCommand  argList
	UpdateCommand  argList
	UpgradeCommand argList
	Verbatim       []string
}

type sourceImageInfoType struct {
	filter   *filter.Filter
	triggers *triggers.Triggers
}

type unpackImageFunction func(client *srpc.Client, streamName, rootDir string,
	buildLog buildLogger) (*sourceImageInfoType, error)

type Builder struct {
	stateDir                  string
	imageServerAddress        string
	logger                    log.Logger
	imageStreamsUrl           string
	streamsLock               sync.RWMutex
	bootstrapStreams          map[string]*bootstrapStream
	imageStreams              map[string]*imageStreamType
	imageStreamsToAutoRebuild []string
	buildResultsLock          sync.RWMutex
	currentBuildLogs          map[string]*bytes.Buffer   // Key: stream name.
	lastBuildResults          map[string]buildResultType // Key: stream name.
	packagerTypes             map[string]packagerType
	variables                 map[string]string
}

func Load(confUrl, variablesFile, stateDir, imageServerAddress string,
	imageRebuildInterval time.Duration, logger log.DebugLogger) (
	*Builder, error) {
	return load(confUrl, variablesFile, stateDir, imageServerAddress,
		imageRebuildInterval, logger)
}

func (b *Builder) BuildImage(streamName string, expiresIn time.Duration,
	gitBranch string, maxSourceAge time.Duration, uploadImage bool,
	logWriter io.Writer) (*image.Image, string, error) {
	return b.buildImage(streamName, expiresIn, gitBranch, maxSourceAge,
		uploadImage, logWriter)
}

func (b *Builder) GetCurrentBuildLog(streamName string) ([]byte, error) {
	return b.getCurrentBuildLog(streamName)
}

func (b *Builder) GetLatestBuildLog(streamName string) ([]byte, error) {
	return b.getLatestBuildLog(streamName)
}

func (b *Builder) ShowImageStream(writer io.Writer, streamName string) {
	b.showImageStream(writer, streamName)
}

func (b *Builder) ShowImageStreams(writer io.Writer) {
	b.showImageStreams(writer)
}

func (b *Builder) WriteHtml(writer io.Writer) {
	b.writeHtml(writer)
}

func BuildImageFromManifest(client *srpc.Client, manifestDir, streamName string,
	expiresIn time.Duration, buildLog buildLogger, logger log.Logger) (
	string, error) {
	_, name, err := buildImageFromManifest(client, manifestDir, streamName,
		expiresIn, true, unpackImageSimple, buildLog)
	return name, err
}

func BuildTreeFromManifest(client *srpc.Client, manifestDir string,
	buildLog *bytes.Buffer, logger log.Logger) (string, error) {
	return buildTreeFromManifest(client, manifestDir, buildLog)
}

func ProcessManifest(manifestDir, rootDir string, buildLog io.Writer) error {
	return processManifest(manifestDir, rootDir, buildLog)
}
