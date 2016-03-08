package filegen

import (
	"sort"
)

func (m *Manager) registerGeneratorForPath(pathname string, gen FileGenerator) {
	m.fileGenerators[pathname] = gen
}

func (m *Manager) getRegisteredPaths() []string {
	pathnames := make([]string, 0, len(m.fileGenerators))
	for pathname := range m.fileGenerators {
		pathnames = append(pathnames, pathname)
	}
	sort.Strings(pathnames)
	return pathnames
}
