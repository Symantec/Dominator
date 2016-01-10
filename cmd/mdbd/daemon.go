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
	"regexp"
	"sort"
	"time"
)

type genericEncoder interface {
	Encode(v interface{}) error
}

func runDaemon(driverFunc driverFunc, url, mdbFileName, hostnameRegex string,
	fetchInterval uint, logger *log.Logger) {
	var prevMdb *mdb.Mdb
	var hostnameRE *regexp.Regexp
	var err error
	if hostnameRegex != ".*" {
		hostnameRE, err = regexp.Compile("^" + hostnameRegex)
		if err != nil {
			logger.Println(err)
			os.Exit(1)
		}
	}
	var cycleStopTime time.Time
	for ; ; sleepUntil(cycleStopTime) {
		cycleStopTime = time.Now().Add(time.Duration(fetchInterval))
		if newMdb := loadMdb(driverFunc, url, logger); newMdb != nil {
			newMdb := selectHosts(newMdb, hostnameRE)
			sort.Sort(newMdb)
			if newMdbIsDifferent(prevMdb, newMdb) {
				if err := writeMdb(newMdb, mdbFileName); err != nil {
					logger.Println(err)
				} else {
					prevMdb = newMdb
				}
			}
		}
	}
}

func sleepUntil(wakeTime time.Time) {
	sleepTime := wakeTime.Sub(time.Now())
	if sleepTime < time.Second {
		sleepTime = time.Second
	}
	time.Sleep(sleepTime)
}

func loadMdb(driverFunc driverFunc, url string, logger *log.Logger) *mdb.Mdb {
	file, err := os.Open(url)
	if err != nil {
		logger.Println("Error opening file " + err.Error())
		return nil
	}
	defer file.Close()
	return driverFunc(bufio.NewReader(file), logger)
}

func selectHosts(inMdb *mdb.Mdb, hostnameRE *regexp.Regexp) *mdb.Mdb {
	if hostnameRE == nil {
		return inMdb
	}
	var outMdb mdb.Mdb
	for _, machine := range inMdb.Machines {
		if hostnameRE.MatchString(machine.Hostname) {
			outMdb.Machines = append(outMdb.Machines, machine)
		}
	}
	return &outMdb
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
