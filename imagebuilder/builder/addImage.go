package builder

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	imageclient "github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem/scanner"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem/util"
	"github.com/Cloud-Foundations/Dominator/lib/filter"
	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/image"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/triggers"
	proto "github.com/Cloud-Foundations/Dominator/proto/imaginator"
)

const timeFormat = "2006-01-02:15:04:05"

var errorTestTimedOut = errors.New("test timed out")

type hasher struct {
	objQ *objectclient.ObjectAdderQueue
}

type testResultType struct {
	buffer   chan byte
	duration time.Duration
	err      error
	prog     string
}

func (h *hasher) Hash(reader io.Reader, length uint64) (
	hash.Hash, error) {
	hash, err := h.objQ.Add(reader, length)
	if err != nil {
		return hash, errors.New("error sending image data: " + err.Error())
	}
	return hash, nil
}

func addImage(client *srpc.Client, request proto.BuildImageRequest,
	img *image.Image) (string, error) {
	if request.ExpiresIn > 0 {
		img.ExpiresAt = time.Now().Add(request.ExpiresIn)
	}
	name := path.Join(request.StreamName, time.Now().Format(timeFormat))
	if err := imageclient.AddImage(client, name, img); err != nil {
		return "", errors.New("remote error: " + err.Error())
	}
	return name, nil
}

func buildFileSystem(client *srpc.Client, dirname string,
	scanFilter *filter.Filter) (
	*filesystem.FileSystem, error) {
	var h hasher
	var err error
	h.objQ, err = objectclient.NewObjectAdderQueue(client)
	if err != nil {
		return nil, err
	}
	fs, err := buildFileSystemWithHasher(dirname, &h, scanFilter)
	if err != nil {
		h.objQ.Close()
		return nil, err
	}
	err = h.objQ.Close()
	if err != nil {
		return nil, err
	}
	return fs, nil
}

func buildFileSystemWithHasher(dirname string, h *hasher,
	scanFilter *filter.Filter) (
	*filesystem.FileSystem, error) {
	fs, err := scanner.ScanFileSystem(dirname, nil, scanFilter, nil, h, nil)
	if err != nil {
		return nil, err
	}
	return &fs.FileSystem, nil
}

func listPackages(rootDir string) ([]image.Package, error) {
	output := new(bytes.Buffer)
	err := runInTarget(nil, output, rootDir, nil, packagerPathname,
		"show-size-multiplier")
	if err != nil {
		return nil, fmt.Errorf("error getting size multiplier: %s", err)
	}
	sizeMultiplier := uint64(1)
	nScanned, err := fmt.Fscanf(output, "%d", &sizeMultiplier)
	if err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf(
				"error decoding size multiplier: %s", err)
		}
	} else if nScanned != 1 {
		return nil, errors.New("malformed size multiplier")
	}
	output.Reset()
	err = runInTarget(nil, output, rootDir, nil, packagerPathname, "list")
	if err != nil {
		return nil, err
	}
	packageMap := make(map[string]image.Package)
	for {
		var name, version string
		var size uint64
		nScanned, err := fmt.Fscanf(output, "%s %s %d\n",
			&name, &version, &size)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if nScanned != 3 {
			return nil, errors.New("malformed line")
		}
		packageMap[name] = image.Package{
			Name:    name,
			Size:    size * sizeMultiplier,
			Version: version,
		}
	}
	packageNames := make([]string, 0, len(packageMap))
	for name := range packageMap {
		packageNames = append(packageNames, name)
	}
	sort.Strings(packageNames)
	var packages []image.Package
	for _, name := range packageNames {
		packages = append(packages, packageMap[name])
	}
	return packages, nil
}

func packImage(client *srpc.Client, request proto.BuildImageRequest,
	dirname string, scanFilter *filter.Filter,
	computedFilesList []util.ComputedFile, imageFilter *filter.Filter,
	trig *triggers.Triggers, buildLog buildLogger) (*image.Image, error) {
	packages, err := listPackages(dirname)
	if err != nil {
		return nil, fmt.Errorf("error listing packages: %s", err)
	}
	buildStartTime := time.Now()
	fs, err := buildFileSystem(client, dirname, scanFilter)
	if err != nil {
		return nil, fmt.Errorf("error building file-system: %s", err)
	}
	if err := util.SpliceComputedFiles(fs, computedFilesList); err != nil {
		return nil, fmt.Errorf("error splicing computed files: %s", err)
	}
	fs.ComputeTotalDataBytes()
	duration := time.Since(buildStartTime)
	speed := uint64(float64(fs.TotalDataBytes) / duration.Seconds())
	fmt.Fprintf(buildLog,
		"Scanned file-system and uploaded %d objects (%s) in %s (%s/s)\n",
		len(fs.InodeTable), format.FormatBytes(fs.TotalDataBytes),
		format.Duration(duration), format.FormatBytes(speed))
	_, oldImage, err := getLatestImage(client, request.StreamName, buildLog)
	if err != nil {
		return nil, fmt.Errorf("error getting latest image: %s", err)
	} else if oldImage != nil {
		patchStartTime := time.Now()
		util.CopyMtimes(oldImage.FileSystem, fs)
		fmt.Fprintf(buildLog, "Copied mtimes in %s\n",
			format.Duration(time.Since(patchStartTime)))
	}
	if err := runTests(dirname, buildLog); err != nil {
		return nil, err
	}
	objClient := objectclient.AttachObjectClient(client)
	// Make a copy of the build log because AddObject() drains the buffer.
	logReader := bytes.NewBuffer(buildLog.Bytes())
	hashVal, _, err := objClient.AddObject(logReader, uint64(logReader.Len()),
		nil)
	if err != nil {
		return nil, err
	}
	if err := objClient.Close(); err != nil {
		return nil, err
	}
	img := &image.Image{
		BuildLog:   &image.Annotation{Object: &hashVal},
		FileSystem: fs,
		Filter:     imageFilter,
		Triggers:   trig,
		Packages:   packages,
	}
	if err := img.Verify(); err != nil {
		return nil, err
	}
	return img, nil
}

func runTests(rootDir string, buildLog buildLogger) error {
	var testProgrammes []string
	err := filepath.Walk(filepath.Join(rootDir, "tests"),
		func(path string, fi os.FileInfo, err error) error {
			if fi == nil || !fi.Mode().IsRegular() || fi.Mode()&0100 == 0 {
				return nil
			}
			testProgrammes = append(testProgrammes, path[len(rootDir):])
			return nil
		})
	if err != nil {
		return err
	}
	if len(testProgrammes) < 1 {
		return nil
	}
	fmt.Fprintf(buildLog, "Running %d tests\n", len(testProgrammes))
	results := make(chan testResultType, 1)
	for _, prog := range testProgrammes {
		go func(prog string) {
			results <- runTest(rootDir, prog)
		}(prog)
	}
	numFailures := 0
	for range testProgrammes {
		result := <-results
		io.Copy(buildLog, &result)
		if result.err != nil {
			fmt.Fprintf(buildLog, "error running: %s: %s\n",
				result.prog, result.err)
			numFailures++
		} else {
			fmt.Fprintf(buildLog, "%s passed in %s\n",
				result.prog, format.Duration(result.duration))
		}
		fmt.Fprintln(buildLog)
	}
	if numFailures > 0 {
		return fmt.Errorf("%d tests failed", numFailures)
	}
	return nil
}

func runTest(rootDir, prog string) testResultType {
	startTime := time.Now()
	result := testResultType{
		buffer: make(chan byte, 4096),
		prog:   prog,
	}
	errChannel := make(chan error, 1)
	timer := time.NewTimer(time.Second * 10)
	go func() {
		errChannel <- runInTarget(nil, &result, rootDir, nil, packagerPathname,
			"run", prog)
	}()
	select {
	case result.err = <-errChannel:
		result.duration = time.Since(startTime)
	case <-timer.C:
		result.err = errorTestTimedOut
	}
	return result
}

func (w *testResultType) Read(p []byte) (int, error) {
	for count := 0; count < len(p); count++ {
		select {
		case p[count] = <-w.buffer:
		default:
			return count, io.EOF
		}
	}
	return len(p), nil
}

func (w *testResultType) Write(p []byte) (int, error) {
	for index, ch := range p {
		select {
		case w.buffer <- ch:
		default:
			return index, io.ErrShortWrite
		}
	}
	return len(p), nil
}
