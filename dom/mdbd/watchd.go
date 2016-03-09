package mdbd

import (
	"encoding/gob"
	"encoding/json"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/mdb"
	"io"
	"log"
	"path"
	"sort"
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
	var lastMdb *mdb.Mdb
	for reader := range fsutil.WatchFile(mdbFileName, logger) {
		mdb := loadFile(reader, mdbFileName, logger)
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
	}
}

func loadFile(reader io.Reader, filename string, logger *log.Logger) *mdb.Mdb {
	decoder := getDecoder(reader, filename)
	var mdb mdb.Mdb
	decodeStartTime := time.Now()
	if err := decoder.Decode(&mdb.Machines); err != nil {
		logger.Printf("Error decoding\t%s\n", err)
		return nil
	}
	sortStartTime := time.Now()
	mdbDecodeTimeDistribution.Add(sortStartTime.Sub(decodeStartTime))
	sort.Sort(&mdb)
	mdbSortTimeDistribution.Add(time.Since(sortStartTime))
	return &mdb
}

func getDecoder(reader io.Reader, filename string) genericDecoder {
	switch path.Ext(filename) {
	case ".gob":
		return gob.NewDecoder(reader)
	default:
		return json.NewDecoder(reader)
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
