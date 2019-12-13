package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	hyperclient "github.com/Cloud-Foundations/Dominator/hypervisor/client"
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

type directoryRestorer struct {
	dirname string
}

type gzipDecompressor struct{}

type tarRestorer struct {
	closer     io.Closer
	nextHeader *tar.Header
	reader     *tar.Reader
}

type tarReader struct {
	reader *tar.Reader
}

type readerMaker interface {
	MakeReader(w io.ReadCloser) io.ReadCloser
}

type vmRestorer interface {
	Close() error
	OpenReader(filename string) (io.ReadCloser, uint64, error)
}

func copyVolumeFromVmRestorer(writer io.Writer, restorer vmRestorer,
	volIndex uint, size uint64, logger log.DebugLogger) error {
	var filename string
	if volIndex == 0 {
		filename = "root"
	} else {
		filename = fmt.Sprintf("secondary-volume.%d", volIndex-1)
	}
	logger.Debugf(0, "uploading %s\n", filename)
	if reader, size, err := restorer.OpenReader(filename); err != nil {
		return err
	} else {
		defer reader.Close()
		startTime := time.Now()
		if _, err := io.CopyN(writer, reader, int64(size)); err != nil {
			return err
		}
		duration := time.Since(startTime)
		speed := uint64(float64(size) / duration.Seconds())
		logger.Debugf(0, "sent %s (%s/s)\n",
			format.FormatBytes(size), format.FormatBytes(speed))
		return nil
	}
}

func decodeJsonFromVmRestorer(restorer vmRestorer, filename string,
	data interface{}) error {
	if reader, size, err := restorer.OpenReader(filename); err != nil {
		return err
	} else {
		defer reader.Close()
		return json.NewDecoder(
			&io.LimitedReader{R: reader, N: int64(size)}).Decode(data)
	}
}

func readFromVmRestorer(restorer vmRestorer, filename string) ([]byte, error) {
	if reader, size, err := restorer.OpenReader(filename); err != nil {
		return nil, err
	} else {
		defer reader.Close()
		data := make([]byte, size)
		if _, err := io.ReadAtLeast(reader, data, int(size)); err != nil {
			return nil, err
		}
		return data, nil
	}
}

func restoreVmSubcommand(args []string, logger log.DebugLogger) error {
	if err := restoreVm(args[0], logger); err != nil {
		return fmt.Errorf("Error restoring VM: %s", err)
	}
	return nil
}

func restoreVm(source string, logger log.DebugLogger) error {
	if hypervisor, err := getHypervisorAddress(); err != nil {
		return err
	} else {
		logger.Debugf(0, "restoring VM on %s\n", hypervisor)
		return restoreVmOnHypervisor(hypervisor, source, logger)
	}
}

func restoreVmOnHypervisor(hypervisor, source string,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	u, err := url.Parse(source)
	if err != nil {
		return err
	}
	var restorer vmRestorer
	if u.Scheme == "dir" {
		if restorer, err = newDirectoryRestorer(u.Path); err != nil {
			return err
		}
	} else if u.Scheme == "file" {
		if strings.HasSuffix(u.Path, ".tar") {
			if restorer, err = newTarRestorer(u.Path, nil); err != nil {
				return err
			}
		} else if strings.HasSuffix(u.Path, ".tar.gz") ||
			strings.HasSuffix(u.Path, ".tgz") {
			restorer, err = newTarRestorer(u.Path, gzipDecompressor{})
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown extension: %s", u.Path)
		}
	} else {
		return fmt.Errorf("unknown scheme: %s", u.Scheme)
	}
	defer restorer.Close()
	logger.Debugln(0, "reading metadata")
	var vmInfo proto.VmInfo
	err = decodeJsonFromVmRestorer(restorer, "info.json", &vmInfo)
	if err != nil {
		return err
	}
	vmInfo.ImageName = ""
	vmInfo.ImageURL = ""
	userData, err := readFromVmRestorer(restorer, "user-data.raw")
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	request := proto.CreateVmRequest{
		DhcpTimeout:          *dhcpTimeout,
		ImageDataSize:        vmInfo.Volumes[0].Size,
		SecondaryVolumes:     vmInfo.Volumes[1:],
		SecondaryVolumesData: true,
		UserDataSize:         uint64(len(userData)),
		VmInfo:               vmInfo,
	}
	conn, err := client.Call("Hypervisor.CreateVm")
	if err != nil {
		return fmt.Errorf("error calling Hypervisor.CreateVm: %s", err)
	}
	doCloseConn := true
	defer func() {
		if doCloseConn {
			conn.Close()
		}
	}()
	if err := conn.Encode(request); err != nil {
		return fmt.Errorf("error encoding request: %s", err)
	}
	if err != nil {
		return err
	}
	err = copyVolumeFromVmRestorer(conn, restorer, 0, vmInfo.Volumes[0].Size,
		logger)
	if err != nil {
		return err
	}
	if _, err := conn.Write(userData); err != nil {
		return err
	}
	for index, volume := range vmInfo.Volumes[1:] {
		err := copyVolumeFromVmRestorer(conn, restorer, uint(index+1),
			volume.Size, logger)
		if err != nil {
			return err
		}
	}
	reply, err := processCreateVmResponses(conn, logger)
	if err != nil {
		return err
	}
	doCloseConn = false
	conn.Close()
	if err := hyperclient.AcknowledgeVm(client, reply.IpAddress); err != nil {
		return fmt.Errorf("error acknowledging VM: %s", err)
	}
	fmt.Println(reply.IpAddress)
	if reply.DhcpTimedOut {
		return errors.New("DHCP ACK timed out")
	}
	if *dhcpTimeout > 0 {
		logger.Debugln(0, "Received DHCP ACK")
	}
	return nil
}

func newDirectoryRestorer(dirname string) (*directoryRestorer, error) {
	if fi, err := os.Stat(dirname); err != nil {
		return nil, err
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dirname)
	}
	return &directoryRestorer{dirname: dirname}, nil
}

func (restorer *directoryRestorer) Close() error {
	return nil
}

func (restorer *directoryRestorer) Filename(filename string) string {
	return filepath.Join(restorer.dirname, filename)
}

func (restorer *directoryRestorer) OpenReader(filename string) (
	io.ReadCloser, uint64, error) {
	file, err := os.OpenFile(restorer.Filename(filename), os.O_RDONLY, 0)
	if err != nil {
		return nil, 0, err
	} else if fi, err := file.Stat(); err != nil {
		file.Close()
		return nil, 0, err
	} else {
		return file, uint64(fi.Size()), nil
	}
}

func (gzipDecompressor) MakeReader(r io.ReadCloser) io.ReadCloser {
	if reader, err := gzip.NewReader(r); err != nil {
		panic(err)
	} else {
		return &wrappedReadCloser{real: r, wrap: reader}
	}
}

func newTarRestorer(filename string,
	decompressor readerMaker) (*tarRestorer, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	readCloser := io.ReadCloser(file)
	if decompressor != nil {
		readCloser = decompressor.MakeReader(readCloser)
	}
	return &tarRestorer{
		closer: readCloser,
		reader: tar.NewReader(readCloser),
	}, nil
}

func (restorer *tarRestorer) Close() error {
	return restorer.closer.Close()
}

func (restorer *tarRestorer) OpenReader(filename string) (
	io.ReadCloser, uint64, error) {
	header := restorer.nextHeader
	if header == nil {
		var err error
		if header, err = restorer.reader.Next(); err != nil {
			return nil, 0, err
		}
	}
	restorer.nextHeader = header
	if header.Name != filename {
		return nil, 0, &os.PathError{
			Op:   "have: " + header.Name + " want:",
			Path: filename,
			Err:  os.ErrNotExist,
		}
	} else {
		restorer.nextHeader = nil
		return ioutil.NopCloser(&tarReader{restorer.reader}),
			uint64(header.Size), nil
	}
}

func (r tarReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}
