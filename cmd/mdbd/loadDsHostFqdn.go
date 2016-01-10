package main

import (
	"encoding/json"
	"github.com/Symantec/Dominator/lib/mdb"
	"io"
	"log"
)

func loadDsHostFqdn(reader io.Reader, logger *log.Logger) *mdb.Mdb {
	type machineType struct {
		Fqdn string
	}

	type dataCentreType map[string]machineType

	type inMdbType map[string]dataCentreType

	var inMdb inMdbType
	var outMdb mdb.Mdb
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&inMdb); err != nil {
		logger.Println("Error decoding: " + err.Error())
		return nil
	}
	for dsName, dataCentre := range inMdb {
		for machineName, inMachine := range dataCentre {
			var outMachine mdb.Machine
			if inMachine.Fqdn == "" {
				outMachine.Hostname = machineName + "." + dsName
			} else {
				outMachine.Hostname = inMachine.Fqdn
			}
			outMdb.Machines = append(outMdb.Machines, outMachine)
		}
	}
	return &outMdb
}
