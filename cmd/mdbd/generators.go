package main

import (
	"bufio"
	"errors"
	"github.com/Symantec/Dominator/lib/mdb"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type driverFunc func(reader io.Reader, datacentre string,
	logger *log.Logger) (*mdb.Mdb, error)

// The generator interface generates an mdb from some source.
type generator interface {
	Generate(datacentre string, logger *log.Logger) (*mdb.Mdb, error)
}

// source implements the generator interface and generates an *mdb.Mdb from
// either a flat file or a url.
type source struct {
	// The function parses the data from url or flat file.
	driverFunc driverFunc
	// the url or path of the flat file
	url string
}

func (s source) Generate(
	datacentre string, logger *log.Logger) (*mdb.Mdb, error) {
	return loadMdb(s.driverFunc, s.url, datacentre, logger)
}

func loadMdb(driverFunc driverFunc, url string, datacentre string,
	logger *log.Logger) (
	*mdb.Mdb, error) {
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

func loadHttpMdb(driverFunc driverFunc, url string, datacentre string,
	logger *log.Logger) (
	*mdb.Mdb, error) {
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
