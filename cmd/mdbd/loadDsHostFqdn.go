package main

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/mdb"
)

func newDsHostFqdnGenerator(args []string,
	logger log.DebugLogger) (generator, error) {
	return sourceGenerator{loadDsHostFqdn, args[0]}, nil
}

func loadDsHostFqdn(reader io.Reader, datacentre string, logger log.Logger) (
	*mdb.Mdb, error) {
	type machineType struct {
		Fqdn string
	}

	type dataCentreType map[string]machineType

	type inMdbType map[string]dataCentreType

	var inMdb inMdbType
	var outMdb mdb.Mdb
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&inMdb); err != nil {
		return nil, errors.New("Error decoding: " + err.Error())
	}
	for dsName, dataCentre := range inMdb {
		if datacentre != "" && dsName != datacentre {
			continue
		}
		for _, inMachine := range dataCentre {
			var outMachine mdb.Machine
			if inMachine.Fqdn != "" {
				outMachine.Hostname = inMachine.Fqdn
				outMdb.Machines = append(outMdb.Machines, outMachine)
			}
		}
	}
	return &outMdb, nil
}
