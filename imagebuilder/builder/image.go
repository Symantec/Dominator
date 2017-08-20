package builder

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/log"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/triggers"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

type manifestType struct {
	SourceImage string
	*filter.Filter
}

func (stream *imageStreamType) build(b *Builder, client *srpc.Client,
	streamName string, expiresIn time.Duration, gitBranch string,
	maxSourceAge time.Duration, buildLog *bytes.Buffer, logger log.Logger) (
	string, error) {
	manifestRoot, err := ioutil.TempDir("",
		strings.Replace(streamName, "/", "_", -1)+".manifest")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(manifestRoot)
	if gitBranch == "" {
		gitBranch = "master"
	}
	variableFunc := b.getVariableFunc(map[string]string{
		"IMAGE_STREAM": streamName,
	})
	fmt.Fprintf(buildLog, "Cloning repository: %s branch: %s\n",
		stream.ManifestUrl, gitBranch)
	startTime := time.Now()
	cmd := exec.Command("git", "clone",
		os.Expand(stream.ManifestUrl, variableFunc), "-b", gitBranch,
		manifestRoot)
	cmd.Stdout = buildLog
	cmd.Stderr = buildLog
	if err := cmd.Run(); err != nil {
		return "", errors.New("error cloning repository: " + err.Error())
	}
	fmt.Fprintf(buildLog, "Cloned repository in %s\n",
		format.Duration(time.Since(startTime)))
	manifestDirectory := os.Expand(stream.ManifestDirectory, variableFunc)
	name, err := buildImageFromManifest(client, streamName,
		path.Join(manifestRoot, manifestDirectory), expiresIn,
		func(client *srpc.Client, streamName, rootDir string,
			logger log.Logger) (string, error) {
			return unpackImage(client, streamName, b, maxSourceAge, rootDir,
				logger)
		}, buildLog, logger)
	if err != nil {
		return "", err
	}
	return name, nil
}

func buildImageFromManifest(client *srpc.Client, streamName, manifestDir string,
	expiresIn time.Duration, unpackImageFunc unpackImageFunction,
	buildLog *bytes.Buffer, logger log.Logger) (string, error) {
	// First load all the various manifest files (fail early on error).
	computedFilesList, err := util.LoadComputedFiles(
		path.Join(manifestDir, "computed-files.json"))
	if os.IsNotExist(err) {
		computedFilesList, err = util.LoadComputedFiles(
			path.Join(manifestDir, "computed-files"))
	}
	if err != nil && !os.IsNotExist(err) {
		return "", errors.New(
			"error loading computed files: " + err.Error())
	}
	imageFilter, err := filter.Load(path.Join(manifestDir, "filter"))
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		imageFilter = &filter.Filter{}
	}
	imageTriggers, err := triggers.Load(path.Join(manifestDir, "triggers"))
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	rootDir, err := ioutil.TempDir("",
		strings.Replace(streamName, "/", "_", -1)+".root")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(rootDir)
	manifest, err := unpackImageAndProcessManifest(client, manifestDir,
		unpackImageFunc, rootDir, buildLog, logger)
	if err != nil {
		return "", err
	}
	startTime := time.Now()
	name, err := addImage(client, streamName, rootDir, manifest.Filter,
		computedFilesList, imageFilter, imageTriggers, expiresIn, buildLog,
		logger)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(buildLog, "Uploaded: %s in %s\n", name,
		format.Duration(time.Since(startTime)))
	return name, nil
}

func buildTreeFromManifest(client *srpc.Client, manifestDir string,
	buildLog *bytes.Buffer, logger log.Logger) (string, error) {
	rootDir, err := ioutil.TempDir("", "tree")
	if err != nil {
		return "", err
	}
	_, err = unpackImageAndProcessManifest(client, manifestDir,
		unpackImageSimple, rootDir, buildLog, logger)
	if err != nil {
		os.RemoveAll(rootDir)
		return "", err
	}
	return rootDir, nil
}

func unpackImageSimple(client *srpc.Client, streamName, rootDir string,
	logger log.Logger) (string, error) {
	return unpackImage(client, streamName, nil, 0, rootDir, logger)
}

func unpackImage(client *srpc.Client, streamName string, builder *Builder,
	maxSourceAge time.Duration, rootDir string,
	logger log.Logger) (string, error) {
	imageName, sourceImage, err := getLatestImage(client, streamName)
	if err != nil {
		return "", err
	}
	if sourceImage == nil {
		return "", errors.New("no images for stream: " + streamName)
	}
	objClient := objectclient.AttachObjectClient(client)
	defer objClient.Close()
	err = util.Unpack(sourceImage.FileSystem, objClient, rootDir, logger)
	if err != nil {
		return "", err
	}
	return imageName, nil
}
