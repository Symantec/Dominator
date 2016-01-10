package main

import (
	"bufio"
	"github.com/Symantec/Dominator/lib/mdb"
	"log"
	"os"
	"strings"
)

func loadFromTextFile(url string, logger *log.Logger) *mdb.Mdb {
	file, err := os.Open(url)
	if err != nil {
		logger.Println("Error opening file " + err.Error())
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var newMdb mdb.Mdb
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 0 {
			var machine mdb.Machine
			machine.Hostname = fields[0]
			if len(fields) > 1 {
				machine.RequiredImage = fields[1]
				if len(fields) > 2 {
					machine.PlannedImage = fields[2]
				}
			}
			newMdb.Machines = append(newMdb.Machines, machine)
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Println(err)
		return nil
	}
	return &newMdb
}
