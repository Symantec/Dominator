package filegen

import (
	"github.com/Symantec/Dominator/lib/hash"
	"sort"
)

func (m *Manager) registerGeneratorForPath(pathname string,
	gen FileGenerator) chan<- struct{} {
	if _, ok := m.pathManagers[pathname]; ok {
		panic(pathname + " already registered")
	}
	notifyChan := make(chan struct{}, 1)
	pathMgr := &pathManager{gen, make(map[string]hash.Hash)}
	m.pathManagers[pathname] = pathMgr
	go m.processPathDataInvalidations(pathname, notifyChan)
	return notifyChan
}

func (m *Manager) processPathDataInvalidations(pathname string,
	ch <-chan struct{}) {
	pathMgr := m.pathManagers[pathname]
	for range ch {
		pathMgr.machineHashes = make(map[string]hash.Hash)
		// TODO(rgooch): Notify RPC clients.
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
