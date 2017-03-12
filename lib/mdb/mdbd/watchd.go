package mdbd

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/Symantec/Dominator/lib/fsutil"
	jsonwriter "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/mdbserver"
	"io"
	"log"
	"os"
	"path"
	"reflect"
	"sort"
	"time"
)

func startMdbDaemon(mdbFileName string, logger *log.Logger) <-chan *mdb.Mdb {
	mdbChannel := make(chan *mdb.Mdb, 1)
	if *mdbServerHostname != "" && *mdbServerPortNum > 0 {
		go serverWatchDaemon(*mdbServerHostname, *mdbServerPortNum, mdbFileName,
			mdbChannel, logger)
	} else {
		go fileWatchDaemon(mdbFileName, mdbChannel, logger)
	}
	return mdbChannel
}

type genericDecoder interface {
	Decode(v interface{}) error
}

func fileWatchDaemon(mdbFileName string, mdbChannel chan<- *mdb.Mdb,
	logger *log.Logger) {
	var lastMdb *mdb.Mdb
	for readCloser := range fsutil.WatchFile(mdbFileName, logger) {
		mdb := loadFile(readCloser, mdbFileName, logger)
		readCloser.Close()
		if mdb == nil {
			continue
		}
		compareStartTime := time.Now()
		if lastMdb == nil || !reflect.DeepEqual(lastMdb, mdb) {
			if lastMdb != nil {
				mdbCompareTimeDistribution.Add(time.Since(compareStartTime))
			}
			mdbChannel <- mdb
			lastMdb = mdb
		}
	}
}

func serverWatchDaemon(mdbServerHostname string, mdbServerPortNum uint,
	mdbFileName string, mdbChannel chan<- *mdb.Mdb, logger *log.Logger) {
	if file, err := os.Open(mdbFileName); err == nil {
		fileMdb := loadFile(file, mdbFileName, logger)
		file.Close()
		if fileMdb != nil {
			sort.Sort(fileMdb)
			mdbChannel <- fileMdb
		}
	}
	address := fmt.Sprintf("%s:%d", mdbServerHostname, mdbServerPortNum)
	for ; ; time.Sleep(time.Second) {
		client, err := srpc.DialHTTP("tcp", address, time.Second*15)
		if err != nil {
			logger.Println(err)
			continue
		}
		conn, err := client.Call("MdbServer.GetMdbUpdates")
		if err != nil {
			logger.Println(err)
			client.Close()
			continue
		}
		decoder := gob.NewDecoder(conn)
		lastMdb := &mdb.Mdb{}
		for {
			var mdbUpdate mdbserver.MdbUpdate
			if err := decoder.Decode(&mdbUpdate); err != nil {
				logger.Println(err)
				break
			} else {
				lastMdb = processUpdate(lastMdb, mdbUpdate)
				sort.Sort(lastMdb)
				mdbChannel <- lastMdb
				if file, err := os.Create(mdbFileName + "~"); err != nil {
					logger.Println(err)
				} else {
					writer := bufio.NewWriter(file)
					var err error
					if isGob(mdbFileName) {
						encoder := gob.NewEncoder(writer)
						err = encoder.Encode(lastMdb.Machines)
					} else {
						err = jsonwriter.WriteWithIndent(writer, "    ",
							lastMdb.Machines)
					}
					if err != nil {
						logger.Println(err)
						os.Remove(mdbFileName + "~")
					} else {
						writer.Flush()
						file.Close()
						os.Rename(mdbFileName+"~", mdbFileName)
					}
				}
			}
		}
		conn.Close()
		client.Close()
	}
}

func loadFile(reader io.Reader, filename string, logger *log.Logger) *mdb.Mdb {
	decoder := getDecoder(reader, filename)
	var mdb mdb.Mdb
	decodeStartTime := time.Now()
	if err := decoder.Decode(&mdb.Machines); err != nil {
		logger.Printf("Error decoding MDB data: %s\n", err)
		return nil
	}
	sortStartTime := time.Now()
	mdbDecodeTimeDistribution.Add(sortStartTime.Sub(decodeStartTime))
	sort.Sort(&mdb)
	mdbSortTimeDistribution.Add(time.Since(sortStartTime))
	return &mdb
}

func isGob(filename string) bool {
	switch path.Ext(filename) {
	case ".gob":
		return true
	default:
		return false
	}
}

func getDecoder(reader io.Reader, filename string) genericDecoder {
	if isGob(filename) {
		return gob.NewDecoder(reader)
	} else {
		return json.NewDecoder(reader)
	}
}

func processUpdate(oldMdb *mdb.Mdb, mdbUpdate mdbserver.MdbUpdate) *mdb.Mdb {
	newMdb := &mdb.Mdb{}
	if len(oldMdb.Machines) < 1 {
		newMdb.Machines = mdbUpdate.MachinesToAdd
		return newMdb
	}
	newMachines := make(map[string]mdb.Machine)
	for _, machine := range oldMdb.Machines {
		newMachines[machine.Hostname] = machine
	}
	for _, machine := range mdbUpdate.MachinesToAdd {
		newMachines[machine.Hostname] = machine
	}
	for _, machine := range mdbUpdate.MachinesToUpdate {
		newMachines[machine.Hostname] = machine
	}
	for _, name := range mdbUpdate.MachinesToDelete {
		delete(newMachines, name)
	}
	newMdb.Machines = make([]mdb.Machine, 0, len(newMachines))
	for _, machine := range newMachines {
		newMdb.Machines = append(newMdb.Machines, machine)
	}
	return newMdb
}
