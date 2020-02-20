package builder

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	stdlog "log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem/util"
	"github.com/Cloud-Foundations/Dominator/lib/filter"
	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/image"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/triggers"
	proto "github.com/Cloud-Foundations/Dominator/proto/imaginator"
)

type gitInfoType struct {
	branch   string
	commitId string
}

func (stream *imageStreamType) build(b *Builder, client *srpc.Client,
	request proto.BuildImageRequest, buildLog buildLogger) (
	*image.Image, error) {
	manifestDirectory, gitInfo, err := stream.getManifest(b, request.StreamName,
		request.GitBranch, request.Variables, buildLog)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(manifestDirectory)
	img, err := buildImageFromManifest(client, manifestDirectory, request,
		b.bindMounts, stream, gitInfo, buildLog)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (stream *imageStreamType) getenv() map[string]string {
	envTable := make(map[string]string, 1)
	envTable["IMAGE_STREAM"] = stream.name
	envTable["IMAGE_STREAM_DIRECTORY_NAME"] = filepath.Dir(stream.name)
	envTable["IMAGE_STREAM_LEAF_NAME"] = filepath.Base(stream.name)
	return envTable
}

func (stream *imageStreamType) getManifest(b *Builder, streamName string,
	gitBranch string, variables map[string]string,
	buildLog io.Writer) (string, *gitInfoType, error) {
	if gitBranch == "" {
		gitBranch = "master"
	}
	variableFunc := b.getVariableFunc(stream.getenv(), variables)
	manifestRoot, err := makeTempDirectory("",
		strings.Replace(streamName, "/", "_", -1)+".manifest")
	if err != nil {
		return "", nil, err
	}
	doCleanup := true
	defer func() {
		if doCleanup {
			os.RemoveAll(manifestRoot)
		}
	}()
	manifestDirectory := os.Expand(stream.ManifestDirectory, variableFunc)
	manifestUrl := os.Expand(stream.ManifestUrl, variableFunc)
	if parsedUrl, err := url.Parse(manifestUrl); err == nil {
		if parsedUrl.Scheme == "dir" {
			if parsedUrl.Path[0] != '/' {
				return "", nil, fmt.Errorf("missing leading slash: %s",
					parsedUrl.Path)
			}
			if gitBranch != "master" {
				return "", nil,
					fmt.Errorf("branch: %s is not master", gitBranch)
			}
			sourceTree := filepath.Join(parsedUrl.Path, manifestDirectory)
			fmt.Fprintf(buildLog, "Copying manifest tree: %s\n", sourceTree)
			if err := fsutil.CopyTree(manifestRoot, sourceTree); err != nil {
				return "", nil, fmt.Errorf("error copying manifest: %s", err)
			}
			doCleanup = false
			return manifestRoot, nil, nil
		}
	}
	fmt.Fprintf(buildLog, "Cloning repository: %s branch: %s\n",
		stream.ManifestUrl, gitBranch)
	err = runCommand(buildLog, "", "git", "init", manifestRoot)
	if err != nil {
		return "", nil, err
	}
	err = runCommand(buildLog, manifestRoot, "git", "remote", "add", "origin",
		manifestUrl)
	if err != nil {
		return "", nil, err
	}
	err = runCommand(buildLog, manifestRoot, "git", "config",
		"core.sparsecheckout", "true")
	if err != nil {
		return "", nil, err
	}
	directorySelector := "*\n"
	if manifestDirectory != "" {
		directorySelector = manifestDirectory + "/*\n"
	}
	err = ioutil.WriteFile(
		filepath.Join(manifestRoot, ".git", "info", "sparse-checkout"),
		[]byte(directorySelector), 0644)
	if err != nil {
		return "", nil, err
	}
	startTime := time.Now()
	err = runCommand(buildLog, manifestRoot, "git", "pull", "--depth=1",
		"origin", gitBranch)
	if err != nil {
		return "", nil, err
	}
	if gitBranch != "master" {
		err = runCommand(buildLog, manifestRoot, "git", "checkout", gitBranch)
		if err != nil {
			return "", nil, err
		}
	}
	loadTime := time.Since(startTime)
	repoSize, err := getTreeSize(manifestRoot)
	if err != nil {
		return "", nil, err
	}
	speed := float64(repoSize) / loadTime.Seconds()
	fmt.Fprintf(buildLog,
		"Downloaded partial repository in %s, size: %s (%s/s)\n",
		format.Duration(loadTime), format.FormatBytes(repoSize),
		format.FormatBytes(uint64(speed)))
	gitDirectory := filepath.Join(manifestRoot, ".git")
	var gitInfo *gitInfoType
	filename := filepath.Join(gitDirectory, "refs", "heads", gitBranch)
	if lines, err := fsutil.LoadLines(filename); err != nil {
		return "", nil, err
	} else if len(lines) != 1 {
		return "", nil, fmt.Errorf("%s does not have only one line", filename)
	} else {
		gitInfo = &gitInfoType{
			branch:   gitBranch,
			commitId: strings.TrimSpace(lines[0]),
		}
	}
	if err := os.RemoveAll(gitDirectory); err != nil {
		return "", nil, err
	}
	if manifestDirectory != "" {
		// Move manifestDirectory into manifestRoot, remove anything else.
		err := os.Rename(filepath.Join(manifestRoot, manifestDirectory),
			gitDirectory)
		if err != nil {
			return "", nil, err
		}
		filenames, err := listDirectory(manifestRoot)
		if err != nil {
			return "", nil, err
		}
		for _, filename := range filenames {
			if filename == ".git" {
				continue
			}
			err := os.RemoveAll(filepath.Join(manifestRoot, filename))
			if err != nil {
				return "", nil, err
			}
		}
		filenames, err = listDirectory(gitDirectory)
		if err != nil {
			return "", nil, err
		}
		for _, filename := range filenames {
			err := os.Rename(filepath.Join(gitDirectory, filename),
				filepath.Join(manifestRoot, filename))
			if err != nil {
				return "", nil, err
			}
		}
		if err := os.Remove(gitDirectory); err != nil {
			return "", nil, err
		}
	}
	doCleanup = false
	return manifestRoot, gitInfo, nil
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

func runCommand(buildLog io.Writer, cwd string, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = cwd
	cmd.Stdout = buildLog
	cmd.Stderr = buildLog
	return cmd.Run()
}

func buildImageFromManifest(client *srpc.Client, manifestDir string,
	request proto.BuildImageRequest, bindMounts []string,
	envGetter environmentGetter, gitInfo *gitInfoType,
	buildLog buildLogger) (*image.Image, error) {
	// First load all the various manifest files (fail early on error).
	computedFilesList, err := util.LoadComputedFiles(
		filepath.Join(manifestDir, "computed-files.json"))
	if os.IsNotExist(err) {
		computedFilesList, err = util.LoadComputedFiles(
			filepath.Join(manifestDir, "computed-files"))
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading computed files: %s", err)
	}
	imageFilter, addFilter, err := loadFilter(manifestDir)
	if err != nil {
		return nil, err
	}
	imageTriggers, addTriggers, err := loadTriggers(manifestDir)
	if err != nil {
		return nil, err
	}
	rootDir, err := makeTempDirectory("",
		strings.Replace(request.StreamName, "/", "_", -1)+".root")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(rootDir)
	fmt.Fprintf(buildLog, "Created image working directory: %s\n", rootDir)
	manifest, err := unpackImageAndProcessManifest(client, manifestDir,
		rootDir, bindMounts, false, envGetter, buildLog)
	if err != nil {
		return nil, err
	}
	if fi, err := os.Lstat(filepath.Join(manifestDir, "tests")); err == nil {
		if fi.IsDir() {
			testsDir := filepath.Join(rootDir, "tests", request.StreamName)
			if err := os.MkdirAll(testsDir, fsutil.DirPerms); err != nil {
				return nil, err
			}
			err := copyFiles(manifestDir, "tests", testsDir, buildLog)
			if err != nil {
				return nil, err
			}
		}
	}
	if addFilter {
		mergeableFilter := &filter.MergeableFilter{}
		mergeableFilter.Merge(manifest.sourceImageInfo.filter)
		mergeableFilter.Merge(imageFilter)
		imageFilter = mergeableFilter.ExportFilter()
	}
	if addTriggers {
		mergeableTriggers := &triggers.MergeableTriggers{}
		mergeableTriggers.Merge(manifest.sourceImageInfo.triggers)
		mergeableTriggers.Merge(imageTriggers)
		imageTriggers = mergeableTriggers.ExportTriggers()
	}
	img, err := packImage(client, request, rootDir, manifest.filter,
		computedFilesList, imageFilter, imageTriggers, buildLog)
	if err != nil {
		return nil, err
	}
	if gitInfo != nil {
		img.BuildBranch = gitInfo.branch
		img.BuildCommitId = gitInfo.commitId
	}
	return img, nil
}

func buildImageFromManifestAndUpload(client *srpc.Client, manifestDir string,
	request proto.BuildImageRequest, bindMounts []string,
	envGetter environmentGetter,
	buildLog buildLogger) (*image.Image, string, error) {
	img, err := buildImageFromManifest(client, manifestDir, request, bindMounts,
		envGetter, nil, buildLog)
	if err != nil {
		return nil, "", err
	}
	name, err := addImage(client, request, img)
	if err != nil {
		return nil, "", err
	}
	return img, name, nil
}

func buildTreeFromManifest(client *srpc.Client, manifestDir string,
	bindMounts []string, envGetter environmentGetter,
	buildLog io.Writer) (string, error) {
	rootDir, err := makeTempDirectory("", "tree")
	if err != nil {
		return "", err
	}
	_, err = unpackImageAndProcessManifest(client, manifestDir, rootDir,
		bindMounts, true, envGetter, buildLog)
	if err != nil {
		os.RemoveAll(rootDir)
		return "", err
	}
	return rootDir, nil
}

func loadFilter(manifestDir string) (*filter.Filter, bool, error) {
	imageFilter, err := filter.Load(filepath.Join(manifestDir, "filter"))
	if err != nil && !os.IsNotExist(err) {
		return nil, false, err
	}
	addFilter, err := filter.Load(filepath.Join(manifestDir, "filter.add"))
	if err != nil && !os.IsNotExist(err) {
		return nil, false, err
	}
	if imageFilter == nil && addFilter == nil {
		return nil, false, nil
	} else if imageFilter != nil && addFilter != nil {
		return nil, false, errors.New(
			"filter and filter.add files both present")
	} else if imageFilter != nil {
		return imageFilter, false, nil
	} else {
		return addFilter, true, nil
	}
}

func loadTriggers(manifestDir string) (*triggers.Triggers, bool, error) {
	imageTriggers, err := triggers.Load(filepath.Join(manifestDir, "triggers"))
	if err != nil && !os.IsNotExist(err) {
		return nil, false, err
	}
	addTriggers, err := triggers.Load(filepath.Join(manifestDir,
		"triggers.add"))
	if err != nil && !os.IsNotExist(err) {
		return nil, false, err
	}
	if imageTriggers == nil && addTriggers == nil {
		return nil, false, nil
	} else if imageTriggers != nil && addTriggers != nil {
		return nil, false, errors.New(
			"triggers and triggers.add files both present")
	} else if imageTriggers != nil {
		return imageTriggers, false, nil
	} else {
		return addTriggers, true, nil
	}
}

func unpackImage(client *srpc.Client, streamName string,
	maxSourceAge, expiresIn time.Duration, rootDir string,
	buildLog io.Writer) (*sourceImageInfoType, error) {
	imageName, sourceImage, err := getLatestImage(client, streamName, buildLog)
	if err != nil {
		return nil, err
	}
	if sourceImage == nil {
		return nil, errors.New(errNoSourceImage + streamName)
	}
	if maxSourceAge > 0 && time.Since(sourceImage.CreatedOn) > maxSourceAge {
		return nil, errors.New(errNoSourceImage + streamName)
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
