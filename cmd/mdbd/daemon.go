package main

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"github.com/Symantec/Dominator/lib/mdb"
	"log"
	"os"
	"path"
	"sort"
	"time"
)

type genericEncoder interface {
	Encode(v interface{}) error
}

func runDaemon(driverFunc driverFunc, url, mdbFileName, zone string,
	fetchInterval uint, logger *log.Logger) {
	var prevMdb *mdb.Mdb
	for {
		cycleStopTime := time.Now().Add(time.Duration(fetchInterval))
		if newMdb := driverFunc(url, logger); newMdb != nil {
			sort.Sort(newMdb)
			if newMdbIsDifferent(prevMdb, newMdb) {
				if err := writeMdb(newMdb, mdbFileName); err != nil {
					logger.Println(err)
				} else {
					prevMdb = newMdb
				}
			}
		}
		sleepTime := cycleStopTime.Sub(time.Now())
		if sleepTime < time.Second {
			sleepTime = time.Second
		}
		time.Sleep(sleepTime)
	}
}

func newMdbIsDifferent(prevMdb, newMdb *mdb.Mdb) bool {
	if prevMdb == nil {
		return true
	}
	if len(prevMdb.Machines) != len(newMdb.Machines) {
		return true
	}
	for index, prevMachine := range prevMdb.Machines {
		if prevMachine != newMdb.Machines[index] {
			return true
		}
	}
	return false
}

func writeMdb(mdb *mdb.Mdb, mdbFileName string) error {
	tmpFileName := mdbFileName + "~"
	file, err := os.Create(tmpFileName)
	if err != nil {
		return errors.New("Error opening file " + err.Error())
	}
	defer os.Remove(tmpFileName)
	defer file.Close()
	encoder := getEncoder(file)
	if err = encoder.Encode(mdb.Machines); err != nil {
		return errors.New("Error encoding " + err.Error())
	}
	return os.Rename(tmpFileName, mdbFileName)
}

func getEncoder(mdbFile *os.File) genericEncoder {
	name := mdbFile.Name()
	name = mdbFile.Name()[:len(name)-1]
	switch path.Ext(name) {
	case ".gob":
		return gob.NewEncoder(mdbFile)
	default:
		return json.NewEncoder(mdbFile)
	}
}
