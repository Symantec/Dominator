package filegen

import ()

func (m *Manager) registerGeneratorForPath(pathname string, gen FileGenerator) {
	m.fileGenerators[pathname] = gen
}
