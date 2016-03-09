package filegen

import (
	"bytes"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/objectserver/memory"
	"log"
	"sort"
	"time"
)

type hashGenerator interface {
	generate(machine mdb.Machine, logger *log.Logger) (
		hashVal hash.Hash, validUntil time.Time, err error)
}

type hashGeneratorWrapper struct {
	dataGenerator FileGenerator
	objectServer  *memory.ObjectServer
}

func (m *Manager) registerDataGeneratorForPath(pathname string,
	gen FileGenerator) chan<- string {
	hashGenerator := &hashGeneratorWrapper{gen, m.objectServer}
	return m.registerHashGeneratorForPath(pathname, hashGenerator)
}

func (m *Manager) registerHashGeneratorForPath(pathname string,
	gen hashGenerator) chan<- string {
	if _, ok := m.pathManagers[pathname]; ok {
		panic(pathname + " already registered")
	}
	notifyChan := make(chan string, 1)
	pathMgr := &pathManager{
		generator:     gen,
		machineHashes: make(map[string]expiringHash)}
	m.pathManagers[pathname] = pathMgr
	go m.processPathDataInvalidations(pathname, notifyChan)
	return notifyChan
}

func (m *Manager) processPathDataInvalidations(pathname string,
	machineNameChannel <-chan string) {
	pathMgr := m.pathManagers[pathname]
	for machineName := range machineNameChannel {
		notification := notificationData{pathname, machineName}
		pathMgr.rwMutex.Lock()
		if machineName == "" {
			pathMgr.machineHashes = make(map[string]expiringHash)
		} else {
			delete(pathMgr.machineHashes, machineName)
		}
		for _, notificationChannel := range m.notifiers {
			notificationChannel <- notification
		}
		pathMgr.rwMutex.Unlock()
	}
}

func (m *Manager) getRegisteredPaths() []string {
	pathnames := make([]string, 0, len(m.pathManagers))
	for pathname := range m.pathManagers {
		pathnames = append(pathnames, pathname)
	}
	sort.Strings(pathnames)
	return pathnames
}

func (g *hashGeneratorWrapper) generate(machine mdb.Machine, logger *log.Logger) (
	hash.Hash, time.Time, error) {
	data, validUntil, err := g.dataGenerator.Generate(machine, logger)
	if err != nil {
		return hash.Hash{}, time.Time{}, err
	}
	hashVal, _, err := g.objectServer.AddObject(bytes.NewReader(data),
		uint64(len(data)), nil)
	if err != nil {
		return hash.Hash{}, time.Time{}, err
	}
	return hashVal, validUntil, nil
}
