package main

import (
	"bufio"
	"errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/mdb"
	"io"
	"net/http"
	"os"
	"strings"
)

type makeGeneratorFunc func([]string) (generator, error)

type sourceDriverFunc func(reader io.Reader, datacentre string,
	logger log.Logger) (*mdb.Mdb, error)

// The generator interface generates an mdb from some source.
type generator interface {
	Generate(datacentre string, logger log.Logger) (*mdb.Mdb, error)
}

func setupGenerators(reader io.Reader, drivers []driver) ([]generator, error) {
	var generators []generator
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 1 || len(fields[0]) < 1 || fields[0][0] == '#' {
			continue
		}
		driverName := fields[0]
		args := fields[1:]
		var drv *driver
		for _, d := range drivers {
			if d.name == driverName {
				drv = &d
				break
			}
		}
		if drv == nil {
			return nil, errors.New("unknown driver: " + driverName)
		}
		if len(args) < drv.minArgs {
			return nil, errors.New("insufficient arguments for: " + driverName)
		}
		if drv.maxArgs >= 0 && len(args) > drv.maxArgs {
			return nil, errors.New("too mant arguments for: " + driverName)
		}
		gen, err := drv.setupFunc(args)
		if err != nil {
			return nil, err
		}
		generators = append(generators, gen)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return generators, nil
}

// sourceGenerator implements the generator interface and generates an *mdb.Mdb
// from either a flat file or a URL.
type sourceGenerator struct {
	driverFunc sourceDriverFunc // Parses the data from URL or flat file.
	url        string           // The URL or path of the flat file.
}

func (s sourceGenerator) Generate(datacentre string, logger log.Logger) (
	*mdb.Mdb, error) {
	return loadMdb(s.driverFunc, s.url, datacentre, logger)
}

func loadMdb(driverFunc sourceDriverFunc, url string, datacentre string,
	logger log.Logger) (*mdb.Mdb, error) {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return loadHttpMdb(driverFunc, url, datacentre, logger)
	}
	file, err := os.Open(url)
	if err != nil {
		return nil, errors.New(("Error opening file " + err.Error()))
	}
	defer file.Close()
	return driverFunc(bufio.NewReader(file), datacentre, logger)
}

func loadHttpMdb(driverFunc sourceDriverFunc, url string, datacentre string,
	logger log.Logger) (*mdb.Mdb, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("HTTP get failed")
	}
	return driverFunc(response.Body, datacentre, logger)
}
