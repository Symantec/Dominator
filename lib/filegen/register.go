package filegen

import (
	"bytes"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/objectserver/memory"
	proto "github.com/Symantec/Dominator/proto/filegenerator"
	"log"
	"sort"
	"time"
)

type hashGenerator interface {
	generate(machine mdb.Machine, logger *log.Logger) (
		hashVal hash.Hash, length uint64, validUntil time.Time, err error)
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
		pathMgr.rwMutex.Lock()
		if machineName == "" {
			for _, mdbData := range m.machineData {
				hashVal, length, validUntil, err := pathMgr.generator.generate(
					mdbData, m.logger)
				if err != nil {
					continue
				}
				pathMgr.machineHashes[mdbData.Hostname] = expiringHash{
					hashVal, length, validUntil}
				files := make([]proto.FileInfo, 1)
				files[0].Pathname = pathname
				files[0].Hash = hashVal
				files[0].Length = length
				yieldResponse := &proto.YieldResponse{mdbData.Hostname, files}
				message := &proto.ServerMessage{YieldResponse: yieldResponse}
				for _, clientChannel := range m.clients {
					clientChannel <- message
				}
				m.scheduleTimer(pathname, mdbData.Hostname, validUntil)
			}
		} else {
			hashVal, length, validUntil, err := pathMgr.generator.generate(
				m.machineData[machineName], m.logger)
			if err != nil {
				continue
			}
			pathMgr.machineHashes[machineName] = expiringHash{
				hashVal, length, validUntil}
			files := make([]proto.FileInfo, 1)
			files[0].Pathname = pathname
			files[0].Hash = hashVal
			files[0].Length = length
			yieldResponse := &proto.YieldResponse{machineName, files}
			message := &proto.ServerMessage{YieldResponse: yieldResponse}
			for _, clientChannel := range m.clients {
				clientChannel <- message
			}
			m.scheduleTimer(pathname, machineName, validUntil)
		}
		pathMgr.rwMutex.Unlock()
	}
}

func (m *Manager) scheduleTimer(pathname string, hostname string,
	validUntil time.Time) {
	if validUntil.IsZero() || time.Now().After(validUntil) {
		return // No expiration or already expired.
	}
	pathMgr := m.pathManagers[pathname]
	time.AfterFunc(validUntil.Sub(time.Now()), func() {
		pathMgr.rwMutex.Lock()
		defer pathMgr.rwMutex.Unlock()
		mdbData, ok := m.machineData[hostname]
		if !ok {
			return
		}
		hashVal, length, validUntil, err := pathMgr.generator.generate(
			mdbData, m.logger)
		if err != nil {
			return
		}
		pathMgr.machineHashes[hostname] = expiringHash{
			hashVal, length, validUntil}
		files := make([]proto.FileInfo, 1)
		files[0].Pathname = pathname
		files[0].Hash = hashVal
		files[0].Length = length
		yieldResponse := &proto.YieldResponse{mdbData.Hostname, files}
		message := &proto.ServerMessage{YieldResponse: yieldResponse}
		for _, clientChannel := range m.clients {
			clientChannel <- message
		}
		m.scheduleTimer(pathname, mdbData.Hostname, validUntil)
	})
}

func (m *Manager) getRegisteredPaths() []string {
	pathnames := make([]string, 0, len(m.pathManagers))
	for pathname := range m.pathManagers {
		pathnames = append(pathnames, pathname)
	}
	sort.Strings(pathnames)
	return pathnames
}

func (g *hashGeneratorWrapper) generate(machine mdb.Machine,
	logger *log.Logger) (
	hash.Hash, uint64, time.Time, error) {
	data, validUntil, err := g.dataGenerator.Generate(machine, logger)
	if err != nil {
		return hash.Hash{}, 0, time.Time{}, err
	}
	length := uint64(len(data))
	hashVal, _, err := g.objectServer.AddObject(bytes.NewReader(data), length,
		nil)
	if err != nil {
		return hash.Hash{}, 0, time.Time{}, err
	}
	return hashVal, length, validUntil, nil
}
