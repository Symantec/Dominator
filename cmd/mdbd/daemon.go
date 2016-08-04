package main

import (
	"bufio"
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/mdb"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

type genericEncoder interface {
	Encode(v interface{}) error
}

func runDaemon(generators []generator, mdbFileName, hostnameRegex string,
	datacentre string, fetchInterval uint, logger *log.Logger, debug bool) {
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
	fetchIntervalDuration := time.Duration(fetchInterval) * time.Second
	for ; ; sleepUntil(cycleStopTime) {
		cycleStopTime = time.Now().Add(fetchIntervalDuration)
		newMdb, err := loadFromAll(generators, datacentre, logger)
		if err != nil {
			logger.Println(err)
			continue
		}
		newMdb = selectHosts(newMdb, hostnameRE)
		sort.Sort(newMdb)
		if newMdbIsDifferent(prevMdb, newMdb) {
			if err := writeMdb(newMdb, mdbFileName); err != nil {
				logger.Println(err)
			} else {
				if debug {
					logger.Printf("Wrote new MDB data, %d machines\n",
						len(newMdb.Machines))
				}
				prevMdb = newMdb
			}
		} else if debug {
			logger.Printf("Refreshed MDB data, same %d machines\n",
				len(newMdb.Machines))
		}
	}
}

func sleepUntil(wakeTime time.Time) {
	runtime.GC() // An opportune time to take out the garbage.
	sleepTime := wakeTime.Sub(time.Now())
	if sleepTime < time.Second {
		sleepTime = time.Second
	}
	time.Sleep(sleepTime)
}

func loadFromAll(generators []generator, datacentre string,
	logger *log.Logger) (*mdb.Mdb, error) {
	machineMap := make(map[string]mdb.Machine)
	for _, gen := range generators {
		mdb, err := gen.Generate(datacentre, logger)
		if err != nil {
			return nil, err
		}
		for _, machine := range mdb.Machines {
			if oldMachine, ok := machineMap[machine.Hostname]; ok {
				oldMachine.UpdateFrom(machine)
				machineMap[machine.Hostname] = oldMachine
			} else {
				machineMap[machine.Hostname] = machine
			}
		}
	}
	var newMdb mdb.Mdb
	for _, machine := range machineMap {
		newMdb.Machines = append(newMdb.Machines, machine)
	}
	return &newMdb, nil
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
	switch path.Ext(mdbFileName) {
	case ".gob":
		if err := gob.NewEncoder(writer).Encode(mdb.Machines); err != nil {
			return err
		}
	default:
		if err := json.WriteWithIndent(writer, "    ",
			mdb.Machines); err != nil {
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	return os.Rename(tmpFileName, mdbFileName)
}
