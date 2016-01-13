package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
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

func runDaemon(sources []source, mdbFileName, hostnameRegex string,
	fetchInterval uint, logger *log.Logger, debug bool) {
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
		if newMdb := loadFromAll(sources, logger); newMdb != nil {
			newMdb := selectHosts(newMdb, hostnameRE)
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
}

func sleepUntil(wakeTime time.Time) {
	runtime.GC() // An opportune time to take out the garbage.
	sleepTime := wakeTime.Sub(time.Now())
	if sleepTime < time.Second {
		sleepTime = time.Second
	}
	time.Sleep(sleepTime)
}

func loadFromAll(sources []source, logger *log.Logger) *mdb.Mdb {
	var newMdb mdb.Mdb
	hostMap := make(map[string]struct{})
	atLeastOneSourceWorked := false
	for _, source := range sources {
		if mdb := loadMdb(source.driverFunc, source.url, logger); mdb != nil {
			atLeastOneSourceWorked = true
			for _, machine := range mdb.Machines {
				if _, ok := hostMap[machine.Hostname]; !ok {
					newMdb.Machines = append(newMdb.Machines, machine)
					hostMap[machine.Hostname] = struct{}{}
				}
			}
		}
	}
	if !atLeastOneSourceWorked {
		return nil
	}
	return &newMdb
}

func loadMdb(driverFunc driverFunc, url string, logger *log.Logger) *mdb.Mdb {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return loadHttpMdb(driverFunc, url, logger)
	}
	file, err := os.Open(url)
	if err != nil {
		logger.Println("Error opening file " + err.Error())
		return nil
	}
	defer file.Close()
	return driverFunc(bufio.NewReader(file), logger)
}

func loadHttpMdb(driverFunc driverFunc, url string,
	logger *log.Logger) *mdb.Mdb {
	response, err := http.Get(url)
	if err != nil {
		logger.Println(err)
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		logger.Println("HTTP get failed: " + err.Error())
		return nil
	}
	return driverFunc(response.Body, logger)
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
