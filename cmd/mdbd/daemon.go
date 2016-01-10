package main

import (
	"bufio"
	"bytes"
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
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	switch path.Ext(mdbFileName) {
	case ".gob":
		if err := gob.NewEncoder(writer).Encode(mdb.Machines); err != nil {
			return err
		}
	default:
		b, err := json.Marshal(mdb.Machines)
		if err != nil {
			return err
		}
		var out bytes.Buffer
		json.Indent(&out, b, "", "    ")
		_, err = out.WriteTo(writer)
		if err != nil {
			return err
		}
		writer.Write([]byte("\n"))
	}
	return os.Rename(tmpFileName, mdbFileName)
}
