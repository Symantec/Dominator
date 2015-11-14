package mdb

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"syscall"
	"time"
)

func startMdbDaemon(mdbFileName string, logger *log.Logger) chan *Mdb {
	mdbChannel := make(chan *Mdb)
	go watchDaemon(mdbFileName, mdbChannel, logger)
	return mdbChannel
}

func watchDaemon(mdbFileName string, mdbChannel chan *Mdb, logger *log.Logger) {
	var lastStat syscall.Stat_t
	var lastMdb *Mdb
	for ; ; time.Sleep(time.Second) {
		var stat syscall.Stat_t
		if err := syscall.Stat(mdbFileName, &stat); err != nil {
			logger.Printf("Error stating file: %s\t%s\n", mdbFileName, err)
			continue
		}
		stat.Atim = lastStat.Atim
		if stat != lastStat {
			file, err := os.Open(mdbFileName)
			if err != nil {
				logger.Printf("Error opening file\t%s\n", err)
				continue
			}
			decoder := json.NewDecoder(file)
			var mdb Mdb
			decodeStartTime := time.Now()
			if err = decoder.Decode(&mdb.Machines); err != nil {
				logger.Printf("Error decoding\t%s\n", err)
				continue
			}
			sortStartTime := time.Now()
			mdbDecodeTimeDistribution.Add(sortStartTime.Sub(decodeStartTime))
			sort.Sort(&mdb)
			compareStartTime := time.Now()
			mdbSortTimeDistribution.Add(compareStartTime.Sub(sortStartTime))
			if lastMdb == nil || !compare(lastMdb, &mdb) {
				if lastMdb != nil {
					mdbCompareTimeDistribution.Add(time.Since(compareStartTime))
				}
				mdbChannel <- &mdb
				lastMdb = &mdb
			}
			lastStat = stat
		}
	}
}

func compare(oldMdb, newMdb *Mdb) bool {
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

func (mdb *Mdb) Less(left, right int) bool {
	if mdb.Machines[left].Hostname < mdb.Machines[right].Hostname {
		return true
	}
	return false
}

func (mdb *Mdb) Swap(left, right int) {
	tmp := mdb.Machines[left]
	mdb.Machines[left] = mdb.Machines[right]
	mdb.Machines[right] = tmp
}
