package main

import (
	"bufio"
	"github.com/Symantec/Dominator/lib/mdb"
	"io"
	"log"
	"strings"
)

func loadText(reader io.Reader, datacentre string, logger *log.Logger) (
	*mdb.Mdb, error) {
	scanner := bufio.NewScanner(reader)
	var newMdb mdb.Mdb
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 0 {
			if fields[0][0] == '#' {
				continue
			}
			var machine mdb.Machine
			machine.Hostname = fields[0]
			if len(fields) > 1 {
				machine.RequiredImage = fields[1]
				if len(fields) > 2 {
					machine.PlannedImage = fields[2]
					if len(fields) > 3 && fields[3] == "true" {
						machine.DisableUpdates = true
					}
				}
			}
			newMdb.Machines = append(newMdb.Machines, machine)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &newMdb, nil
}
