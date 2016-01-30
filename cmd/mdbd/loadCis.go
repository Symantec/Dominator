package main

import (
	"encoding/json"
	"errors"
	"github.com/Symantec/Dominator/lib/mdb"
	"io"
	"log"
)

func loadCis(reader io.Reader, datacentre string, logger *log.Logger) (
	*mdb.Mdb, error) {

	type instanceMetadataType struct {
		RequiredImage string `json:"required_image"`
		PlannedImage  string `json:"planned_image"`
	}

	type sourceType struct {
		Name             string
		InstanceMetadata instanceMetadataType `json:"instance_metadata"`
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
		if hit.Source.InstanceMetadata.RequiredImage != "" {
			outMachine.RequiredImage = hit.Source.InstanceMetadata.RequiredImage
		}
		if hit.Source.InstanceMetadata.PlannedImage != "" {
			outMachine.PlannedImage = hit.Source.InstanceMetadata.PlannedImage
		}
		outMdb.Machines = append(outMdb.Machines, outMachine)
	}
	return &outMdb, nil
}
