package images

import (
	"time"

	"github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/stringutil"
)

func newManager(imageServerAddress string, logger log.Logger) *Manager {
	imageInterestChannel := make(chan map[string]struct{})
	imageRequestChannel := make(chan string)
	imageExpireChannel := make(chan string, 16)
	m := &Manager{
		imageServerAddress:   imageServerAddress,
		logger:               logger,
		deduper:              stringutil.NewStringDeduplicator(false),
		imageInterestChannel: imageInterestChannel,
		imageRequestChannel:  imageRequestChannel,
		imageExpireChannel:   imageExpireChannel,
		imagesByName:         make(map[string]*image.Image),
		missingImages:        make(map[string]error),
	}
	go m.manager(imageInterestChannel, imageRequestChannel, imageExpireChannel)
	return m
}

func (m *Manager) getNoWait(name string) (*image.Image, error) {
	m.RLock()
	defer m.RUnlock()
	if image := m.imagesByName[name]; image != nil {
		return image, nil
	}
	if err, ok := m.missingImages[name]; ok {
		return nil, err
	}
	return nil, nil
}

func (m *Manager) getWait(name string) (*image.Image, error) {
	if image, err := m.getNoWait(name); err != nil {
		return nil, err
	} else if image != nil {
		return image, nil
	}
	m.imageRequestChannel <- name
	m.imageRequestChannel <- ""
	return m.getNoWait(name)
}

func (m *Manager) setImageInterestList(images map[string]struct{}, wait bool) {
	delete(images, "")
	m.imageInterestChannel <- images
	if wait {
		m.imageRequestChannel <- ""
	}
}

func (m *Manager) manager(imageInterestChannel <-chan map[string]struct{},
	imageRequestChannel <-chan string,
	imageExpireChannel <-chan string) {
	var imageClient *srpc.Client
	timer := time.NewTimer(time.Second)
	for {
		select {
		case imageList := <-imageInterestChannel:
			imageClient = m.setInterest(imageClient, imageList)
		case name := <-imageRequestChannel:
			if name == "" {
				continue
			}
			imageClient = m.requestImage(imageClient, name)
		case name := <-imageExpireChannel:
			m.Lock()
			delete(m.imagesByName, name)
			m.missingImages[name] = nil // Try to get it again (expire extended)
			m.Unlock()
			m.rebuildDeDuper()
		case <-timer.C:
			// Loop over missing (pending) images. First obtain a copy.
			missingImages := make(map[string]struct{})
			for name := range m.missingImages {
				missingImages[name] = struct{}{}
			}
			for name := range missingImages {
				imageClient = m.requestImage(imageClient, name)
			}
		}
		if len(m.missingImages) > 0 {
			timer.Reset(time.Second)
		}
	}
}

func (m *Manager) setInterest(imageClient *srpc.Client,
	imageList map[string]struct{}) *srpc.Client {
	for name := range imageList {
		imageClient = m.requestImage(imageClient, name)
	}
	deletedSome := false
	// Clean up unreferenced images.
	for name := range m.imagesByName {
		if _, ok := imageList[name]; !ok {
			m.Lock()
			delete(m.imagesByName, name)
			m.Unlock()
			deletedSome = true
		}
	}
	for name := range m.missingImages {
		if _, ok := imageList[name]; !ok {
			m.Lock()
			delete(m.missingImages, name)
			m.Unlock()
		}
	}
	if deletedSome {
		m.rebuildDeDuper()
	}
	return imageClient
}

func (m *Manager) requestImage(imageClient *srpc.Client,
	name string) *srpc.Client {
	if _, ok := m.imagesByName[name]; ok {
		return imageClient
	}
	var img *image.Image
	var err error
	imageClient, img, err = m.loadImage(imageClient, name)
	m.Lock()
	defer m.Unlock()
	if img != nil && err == nil {
		delete(m.missingImages, name)
		m.imagesByName[name] = img
		return imageClient
	}
	delete(m.imagesByName, name)
	m.missingImages[name] = err
	return imageClient
}

func (m *Manager) loadImage(imageClient *srpc.Client, name string) (
	*srpc.Client, *image.Image, error) {
	if imageClient == nil {
		var err error
		imageClient, err = srpc.DialHTTP("tcp", m.imageServerAddress, 0)
		if err != nil {
			if !m.loggedDialFailure {
				m.logger.Printf("Error dialing: %s: %s\n",
					m.imageServerAddress, err)
				m.loggedDialFailure = true
			}
			return nil, nil, err
		}
	}
	img, err := client.GetImage(imageClient, name)
	if err != nil {
		m.logger.Printf("Error calling: %s\n", err)
		imageClient.Close()
		return nil, nil, err
	}
	if img == nil || m.scheduleExpiration(img, name) {
		return imageClient, nil, nil
	}
	if err := img.FileSystem.RebuildInodePointers(); err != nil {
		m.logger.Printf("Error building inode pointers for image: %s %s",
			name, err)
		return imageClient, nil, err
	}
	img.ReplaceStrings(m.deduper.DeDuplicate)
	img.FileSystem = img.FileSystem.Filter(img.Filter) // Apply filter.
	// Build cache data now to avoid potential concurrent builds later.
	img.FileSystem.InodeToFilenamesTable()
	img.FileSystem.FilenameToInodeTable()
	img.FileSystem.HashToInodesTable()
	img.FileSystem.ComputeTotalDataBytes()
	img.FileSystem.BuildEntryMap()
	m.logger.Printf("Got image: %s\n", name)
	return imageClient, img, nil
}

func (m *Manager) rebuildDeDuper() {
	m.deduper.Clear()
	for _, image := range m.imagesByName {
		image.ReplaceStrings(m.deduper.DeDuplicate)
	}
}

func (m *Manager) scheduleExpiration(image *image.Image, name string) bool {
	if image.ExpiresAt.IsZero() {
		return false
	}
	duration := image.ExpiresAt.Sub(time.Now())
	if duration <= 0 {
		return true
	}
	time.AfterFunc(duration, func() {
		m.logger.Printf("Auto expiring (deleting) image: %s\n", name)
		m.imageExpireChannel <- name
	})
	return false
}
