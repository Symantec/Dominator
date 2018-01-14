package builder

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/format"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/triggers"
)

func (stream *imageStreamType) build(b *Builder, client *srpc.Client,
	streamName string, expiresIn time.Duration, gitBranch string,
	maxSourceAge time.Duration, buildLog *bytes.Buffer) (
	string, error) {
	manifestDirectory, err := stream.getManifest(b, streamName,
		gitBranch, buildLog)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(manifestDirectory)
	name, err := buildImageFromManifest(client, streamName, manifestDirectory,
		expiresIn,
		func(client *srpc.Client, streamName, rootDir string,
			buildLog *bytes.Buffer) (*sourceImageInfoType, error) {
			return unpackImage(client, streamName, b, maxSourceAge, expiresIn,
				rootDir, buildLog)
		}, buildLog)
	if err != nil {
		return "", err
	}
	return name, nil
}

func (stream *imageStreamType) getManifest(b *Builder, streamName string,
	gitBranch string, buildLog *bytes.Buffer) (string, error) {
	if gitBranch == "" {
		gitBranch = "master"
	}
	variableFunc := b.getVariableFunc(map[string]string{
		"IMAGE_STREAM": streamName,
	})
	fmt.Fprintf(buildLog, "Cloning repository: %s branch: %s\n",
		stream.ManifestUrl, gitBranch)
	manifestRoot, err := ioutil.TempDir("",
		strings.Replace(streamName, "/", "_", -1)+".manifest")
	if err != nil {
		return "", err
	}
	doCleanup := true
	defer func() {
		if doCleanup {
			os.RemoveAll(manifestRoot)
		}
	}()
	manifestDirectory := os.Expand(stream.ManifestDirectory, variableFunc)
	manifestUrl := os.Expand(stream.ManifestUrl, variableFunc)
	err = runCommand(buildLog, "", "git", "init", manifestRoot)
	if err != nil {
		return "", err
	}
	err = runCommand(buildLog, manifestRoot, "git", "remote", "add", "origin",
		manifestUrl)
	if err != nil {
		return "", err
	}
	err = runCommand(buildLog, manifestRoot, "git", "config",
		"core.sparsecheckout", "true")
	if err != nil {
		return "", err
	}
	directorySelector := "*\n"
	if manifestDirectory != "" {
		directorySelector = manifestDirectory + "/*\n"
	}
	err = ioutil.WriteFile(
		path.Join(manifestRoot, ".git", "info", "sparse-checkout"),
		[]byte(directorySelector), 0644)
	if err != nil {
		return "", err
	}
	startTime := time.Now()
	err = runCommand(buildLog, manifestRoot, "git", "pull", "--depth=1",
		"origin", gitBranch)
	if err != nil {
		return "", err
	}
	if gitBranch != "master" {
		err = runCommand(buildLog, manifestRoot, "git", "checkout", gitBranch)
		if err != nil {
			return "", err
		}
	}
	loadTime := time.Since(startTime)
	repoSize, err := getTreeSize(manifestRoot)
	if err != nil {
		return "", err
	}
	speed := float64(repoSize) / loadTime.Seconds()
	fmt.Fprintf(buildLog,
		"Downloaded partial repository in %s, size: %s (%s/s)\n",
		format.Duration(loadTime), format.FormatBytes(repoSize),
		format.FormatBytes(uint64(speed)))
	gitDirectory := path.Join(manifestRoot, ".git")
	if err := os.RemoveAll(gitDirectory); err != nil {
		return "", err
	}
	if manifestDirectory != "" {
		// Move manifestDirectory into manifestRoot, remove anything else.
		err := os.Rename(path.Join(manifestRoot, manifestDirectory),
			gitDirectory)
		if err != nil {
			return "", err
		}
		filenames, err := listDirectory(manifestRoot)
		if err != nil {
			return "", err
		}
		for _, filename := range filenames {
			if filename == ".git" {
				continue
			}
			err := os.RemoveAll(path.Join(manifestRoot, filename))
			if err != nil {
				return "", err
			}
		}
		filenames, err = listDirectory(gitDirectory)
		if err != nil {
			return "", err
		}
		for _, filename := range filenames {
			err := os.Rename(path.Join(gitDirectory, filename),
				path.Join(manifestRoot, filename))
			if err != nil {
				return "", err
			}
		}
		if err := os.Remove(gitDirectory); err != nil {
			return "", err
		}
	}
	doCleanup = false
	return manifestRoot, nil
}

func getTreeSize(dirname string) (uint64, error) {
	var size uint64
	err := filepath.Walk(dirname,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			size += uint64(info.Size())
			return nil
		})
	if err != nil {
		return 0, err
	}
	return size, nil
}

func listDirectory(directoryName string) ([]string, error) {
	directory, err := os.Open(directoryName)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	filenames, err := directory.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	return filenames, nil
}

func runCommand(buildLog *bytes.Buffer, cwd string, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = cwd
	cmd.Stdout = buildLog
	cmd.Stderr = buildLog
	return cmd.Run()
}

func buildImageFromManifest(client *srpc.Client, streamName, manifestDir string,
	expiresIn time.Duration, unpackImageFunc unpackImageFunction,
	buildLog *bytes.Buffer) (string, error) {
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
	fmt.Fprintf(buildLog, "Created image working directory: %s\n", rootDir)
	manifest, err := unpackImageAndProcessManifest(client, manifestDir,
		unpackImageFunc, rootDir, buildLog)
	if err != nil {
		return "", err
	}
	startTime := time.Now()
	name, err := addImage(client, streamName, rootDir, manifest.filter,
		computedFilesList, imageFilter, imageTriggers, expiresIn, buildLog)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(buildLog, "Uploaded: %s in %s\n", name,
		format.Duration(time.Since(startTime)))
	return name, nil
}

func buildTreeFromManifest(client *srpc.Client, manifestDir string,
	buildLog *bytes.Buffer) (string, error) {
	rootDir, err := ioutil.TempDir("", "tree")
	if err != nil {
		return "", err
	}
	_, err = unpackImageAndProcessManifest(client, manifestDir,
		unpackImageSimple, rootDir, buildLog)
	if err != nil {
		os.RemoveAll(rootDir)
		return "", err
	}
	return rootDir, nil
}

func unpackImageSimple(client *srpc.Client, streamName, rootDir string,
	buildLog *bytes.Buffer) (*sourceImageInfoType, error) {
	return unpackImage(client, streamName, nil, 0, 0, rootDir, buildLog)
}

func unpackImage(client *srpc.Client, streamName string, builder *Builder,
	maxSourceAge, expiresIn time.Duration, rootDir string,
	buildLog *bytes.Buffer) (*sourceImageInfoType, error) {
	imageName, sourceImage, err := getLatestImage(client, streamName, buildLog)
	if err != nil {
		return nil, err
	}
	if sourceImage == nil {
		if builder == nil {
			return nil, errors.New("no images for stream: " + streamName)
		}
		fmt.Fprintf(buildLog, "No source image: %s, attempting to build one\n",
			streamName)
		imageName, _, err = builder.build(client, streamName, expiresIn,
			"master", maxSourceAge)
		if err != nil {
			return nil, err
		}
		sourceImage, err = getImage(client, imageName, buildLog)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(buildLog, "Built new source image: %s\n", imageName)
		sourceImage.FileSystem.RebuildInodePointers()
	}
	if maxSourceAge > 0 && time.Since(sourceImage.CreatedOn) > maxSourceAge &&
		builder != nil {
		fmt.Fprintf(buildLog,
			"Image: %s is too old, attempting to build a new one\n",
			imageName)
		imageName, _, err = builder.build(client, streamName, expiresIn,
			"master", maxSourceAge)
		if err != nil {
			return nil, err
		}
		sourceImage, err = getImage(client, imageName, buildLog)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(buildLog, "Built new source image: %s\n", imageName)
		sourceImage.FileSystem.RebuildInodePointers()
	}
	objClient := objectclient.AttachObjectClient(client)
	defer objClient.Close()
	err = util.Unpack(sourceImage.FileSystem, objClient, rootDir,
		stdlog.New(buildLog, "", 0))
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(buildLog, "Source image: %s\n", imageName)
	return &sourceImageInfoType{sourceImage.Filter, sourceImage.Triggers}, nil
}
