package scanner

import (
	"errors"
)

func (imdb *ImageDataBase) addImage(image *Image, name string) error {
	imdb.Lock()
	defer imdb.Unlock()
	if _, ok := imdb.imageMap[name]; ok {
		return errors.New("image: " + name + " already exists")
	} else {
		imdb.imageMap[name] = image
		return nil
	}
}

func (imdb *ImageDataBase) checkImage(name string) bool {
	imdb.RLock()
	defer imdb.RUnlock()
	_, ok := imdb.imageMap[name]
	return ok
}

func (imdb *ImageDataBase) deleteImage(name string) error {
	imdb.Lock()
	defer imdb.Unlock()
	if _, ok := imdb.imageMap[name]; ok {
		delete(imdb.imageMap, name)
		return nil
	} else {
		return errors.New("image: " + name + " does not exist")
	}
}

func (imdb *ImageDataBase) getImage(name string) *Image {
	imdb.RLock()
	defer imdb.RUnlock()
	return imdb.imageMap[name]
}

func (imdb *ImageDataBase) listImages() []string {
	imdb.RLock()
	defer imdb.RUnlock()
	names := make([]string, 0)
	for name, _ := range imdb.imageMap {
		names = append(names, name)
	}
	return names
}
