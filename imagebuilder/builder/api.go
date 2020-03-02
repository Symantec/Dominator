package builder

import (
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem/util"
	"github.com/Cloud-Foundations/Dominator/lib/filter"
	"github.com/Cloud-Foundations/Dominator/lib/image"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/slavedriver"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/triggers"
	proto "github.com/Cloud-Foundations/Dominator/proto/imaginator"
)

type buildLogger interface {
	Bytes() []byte
	io.Writer
}

type environmentGetter interface {
	getenv() map[string]string
}

type imageBuilder interface {
	build(b *Builder, client *srpc.Client, request proto.BuildImageRequest,
		buildLog buildLogger) (*image.Image, error)
}

type bootstrapStream struct {
	builder          *Builder
	name             string
	BootstrapCommand []string
	*filter.Filter
	imageFilter      *filter.Filter
	ImageFilterUrl   string
	imageTriggers    *triggers.Triggers
	ImageTriggersUrl string
	PackagerType     string
}

type buildResultType struct {
	imageName  string
	startTime  time.Time
	finishTime time.Time
	buildLog   []byte
	error      error
}

type masterConfigurationType struct {
	BindMounts                []string                    `json:",omitempty"`
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
	BuilderGroups     []string
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
	computedFiles []util.ComputedFile
	filter        *filter.Filter
	triggers      *triggers.Triggers
}

type Builder struct {
	bindMounts                []string
	stateDir                  string
	imageServerAddress        string
	logger                    log.Logger
	imageStreamsUrl           string
	streamsLock               sync.RWMutex
	bootstrapStreams          map[string]*bootstrapStream
	imageStreams              map[string]*imageStreamType
	imageStreamsToAutoRebuild []string
	slaveDriver               *slavedriver.SlaveDriver
	buildResultsLock          sync.RWMutex
	currentBuildLogs          map[string]*bytes.Buffer   // Key: stream name.
	lastBuildResults          map[string]buildResultType // Key: stream name.
	packagerTypes             map[string]packagerType
	variables                 map[string]string
}

func Load(confUrl, variablesFile, stateDir, imageServerAddress string,
	imageRebuildInterval time.Duration, slaveDriver *slavedriver.SlaveDriver,
	logger log.DebugLogger) (*Builder, error) {
	return load(confUrl, variablesFile, stateDir, imageServerAddress,
		imageRebuildInterval, slaveDriver, logger)
}

func (b *Builder) BuildImage(request proto.BuildImageRequest,
	authInfo *srpc.AuthInformation,
	logWriter io.Writer) (*image.Image, string, error) {
	return b.buildImage(request, authInfo, logWriter)
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
	expiresIn time.Duration, bindMounts []string, buildLog buildLogger,
	logger log.Logger) (
	string, error) {
	_, name, err := buildImageFromManifestAndUpload(client, manifestDir,
		proto.BuildImageRequest{
			StreamName: streamName,
			ExpiresIn:  expiresIn,
		},
		bindMounts, nil, buildLog)
	return name, err
}

func BuildTreeFromManifest(client *srpc.Client, manifestDir string,
	bindMounts []string, buildLog io.Writer,
	logger log.Logger) (string, error) {
	return buildTreeFromManifest(client, manifestDir, bindMounts, nil, buildLog)
}

func ProcessManifest(manifestDir, rootDir string, bindMounts []string,
	buildLog io.Writer) error {
	return processManifest(manifestDir, rootDir, bindMounts, nil, buildLog)
}

func UnpackImageAndProcessManifest(client *srpc.Client, manifestDir string,
	rootDir string, bindMounts []string, buildLog io.Writer) error {
	_, err := unpackImageAndProcessManifest(client, manifestDir, rootDir,
		bindMounts, true, nil, buildLog)
	return err
}
