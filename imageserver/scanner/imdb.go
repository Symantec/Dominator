package scanner

import (
	"bufio"
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/image"
	"os"
	"path"
)

func (imdb *ImageDataBase) addImage(image *image.Image, name string) error {
	imdb.Lock()
	defer imdb.Unlock()
	if _, ok := imdb.imageMap[name]; ok {
		return errors.New("image: " + name + " already exists")
	} else {
		filename := path.Join(imdb.baseDir, name)
		err := os.MkdirAll(path.Dir(filename), 0755)
		if err != nil {
			return err
		}
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()
		writer := bufio.NewWriter(file)
		defer writer.Flush()
		encoder := gob.NewEncoder(writer)
		encoder.Encode(image)
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
		filename := path.Join(imdb.baseDir, name)
		err := os.Remove(filename)
		if err != nil {
			return err
		}
		delete(imdb.imageMap, name)
		return nil
	} else {
		return errors.New("image: " + name + " does not exist")
	}
}

func (imdb *ImageDataBase) getImage(name string) *image.Image {
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
