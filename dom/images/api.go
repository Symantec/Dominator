package images

import (
	"sync"

	"github.com/Cloud-Foundations/Dominator/lib/image"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/stringutil"
)

type Manager struct {
	imageServerAddress string
	logger             log.Logger
	loggedDialFailure  bool
	sync.RWMutex
	deduper *stringutil.StringDeduplicator
	// Protected by lock.
	imageInterestChannel chan<- map[string]struct{}
	imageRequestChannel  chan<- string
	imageExpireChannel   chan<- string
	imagesByName         map[string]*image.Image
	missingImages        map[string]error
}

func New(imageServerAddress string, logger log.Logger) *Manager {
	return newManager(imageServerAddress, logger)
}

func (m *Manager) Get(name string, wait bool) (*image.Image, error) {
	if name == "" {
		return nil, nil
	}
	if wait {
		return m.getWait(name)
	}
	return m.getNoWait(name)
}

func (m *Manager) GetNoError(name string) *image.Image {
	if name == "" {
		return nil
	}
	img, _ := m.getNoWait(name)
	return img
}

func (m *Manager) SetImageInterestList(images map[string]struct{}, wait bool) {
	m.setImageInterestList(images, wait)
}

func (m *Manager) String() string {
	return m.imageServerAddress
}
