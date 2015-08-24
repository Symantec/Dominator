package mdb

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"syscall"
	"time"
)

func startMdbDaemon(mdbFileName string) chan *Mdb {
	mdbChannel := make(chan *Mdb)
	go watchDaemon(mdbFileName, mdbChannel)
	return mdbChannel
}

func watchDaemon(mdbFileName string, mdbChannel chan *Mdb) {
	var lastStat syscall.Stat_t
	var lastMdb *Mdb
	for ; ; time.Sleep(time.Second) {
		var stat syscall.Stat_t
		err := syscall.Stat(mdbFileName, &stat)
		if err != nil {
			fmt.Printf("Error stating file: %s\t%s\n", mdbFileName, err)
			continue
		}
		if stat != lastStat {
			file, err := os.Open(mdbFileName)
			if err != nil {
				fmt.Printf("Error opening file\t%s\n", err)
				continue
			}
			decoder := json.NewDecoder(file)
			var mdb Mdb
			err = decoder.Decode(&mdb.Machines)
			if err != nil {
				fmt.Printf("Error decoding\t%s\n", err)
				continue
			}
			sort.Sort(&mdb)
			if lastMdb == nil || !compare(lastMdb, &mdb) {
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
