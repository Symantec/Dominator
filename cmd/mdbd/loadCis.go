package main

import (
	"encoding/json"
	"errors"
	"github.com/Symantec/Dominator/lib/mdb"
	"io"
	"log"
)

func loadCis(reader io.Reader, logger *log.Logger) (*mdb.Mdb, error) {
	type sourceType struct {
		Name string
	}

	type hitType struct {
		Source sourceType `json:"_source"`
	}

	type hitListType struct {
		Hits []hitType
	}

	type inMdbType struct {
		Hits hitListType
	}

	var inMdb inMdbType
	var outMdb mdb.Mdb
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&inMdb); err != nil {
		return nil, errors.New("Error decoding: " + err.Error())
	}
	for _, hit := range inMdb.Hits.Hits {
		var outMachine mdb.Machine
		outMachine.Hostname = hit.Source.Name
		outMdb.Machines = append(outMdb.Machines, outMachine)
	}
	return &outMdb, nil
}
