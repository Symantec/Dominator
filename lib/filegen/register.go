package filegen

import (
	"sort"
)

func (m *Manager) registerGeneratorForPath(pathname string,
	gen FileGenerator) chan<- string {
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
