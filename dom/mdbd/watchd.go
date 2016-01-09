package mdbd

import (
	"encoding/gob"
	"encoding/json"
	"github.com/Symantec/Dominator/lib/mdb"
	"log"
	"os"
	"path"
	"sort"
	"syscall"
	"time"
)

func startMdbDaemon(mdbFileName string, logger *log.Logger) <-chan *mdb.Mdb {
	mdbChannel := make(chan *mdb.Mdb)
	go watchDaemon(mdbFileName, mdbChannel, logger)
	return mdbChannel
}

type genericDecoder interface {
	Decode(v interface{}) error
}

func watchDaemon(mdbFileName string, mdbChannel chan<- *mdb.Mdb,
	logger *log.Logger) {
	var lastStat syscall.Stat_t
	var lastMdb *mdb.Mdb
	for ; ; time.Sleep(time.Second) {
		var stat syscall.Stat_t
		if err := syscall.Stat(mdbFileName, &stat); err != nil {
			logger.Printf("Error stating file: %s\t%s\n", mdbFileName, err)
			continue
		}
		stat.Atim = lastStat.Atim
		if stat != lastStat {
			mdb := loadFile(mdbFileName, logger)
			if mdb == nil {
				continue
			}
			compareStartTime := time.Now()
			if lastMdb == nil || !compare(lastMdb, mdb) {
				if lastMdb != nil {
					mdbCompareTimeDistribution.Add(time.Since(compareStartTime))
				}
				mdbChannel <- mdb
				lastMdb = mdb
			}
			lastStat = stat
		}
	}
}

func loadFile(mdbFileName string, logger *log.Logger) *mdb.Mdb {
	file, err := os.Open(mdbFileName)
	if err != nil {
		logger.Printf("Error opening file\t%s\n", err)
		return nil
	}
	decoder := getDecoder(file)
	var mdb mdb.Mdb
	decodeStartTime := time.Now()
	if err = decoder.Decode(&mdb.Machines); err != nil {
		logger.Printf("Error decoding\t%s\n", err)
		return nil
	}
	sortStartTime := time.Now()
	mdbDecodeTimeDistribution.Add(sortStartTime.Sub(decodeStartTime))
	sort.Sort(&mdb)
	mdbSortTimeDistribution.Add(time.Since(sortStartTime))
	return &mdb
}

func getDecoder(mdbFile *os.File) genericDecoder {
	switch path.Ext(mdbFile.Name()) {
	case ".gob":
		return gob.NewDecoder(mdbFile)
	default:
		return json.NewDecoder(mdbFile)
	}
}

func compare(oldMdb, newMdb *mdb.Mdb) bool {
	if len(oldMdb.Machines) != len(newMdb.Machines) {
		return false
	}
	for index, oldMachine := range oldMdb.Machines {
		if oldMachine != newMdb.Machines[index] {
			return false
		}
	}
	return true
}
