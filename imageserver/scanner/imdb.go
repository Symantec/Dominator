package scanner

import (
	"bufio"
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/image"
	"log"
	"os"
	"path"
)

func (imdb *ImageDataBase) addImage(image *image.Image, name string) error {
	if err := verifyImage(image); err != nil {
		return err
	}
	imdb.Lock()
	defer imdb.Unlock()
	if _, ok := imdb.imageMap[name]; ok {
		return errors.New("image: " + name + " already exists")
	} else {
		filename := path.Join(imdb.baseDir, name)
		if err := os.MkdirAll(path.Dir(filename), 0755); err != nil {
			return err
		}
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()
		w := bufio.NewWriter(file)
		defer w.Flush()
		writer := fsutil.NewChecksumWriter(w)
		defer writer.WriteChecksum()
		encoder := gob.NewEncoder(writer)
		encoder.Encode(image)
		imdb.imageMap[name] = image
		imdb.addNotifiers.send(name, "add", imdb.logger)
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
		if err := os.Remove(filename); err != nil {
			return err
		}
		delete(imdb.imageMap, name)
		imdb.deleteNotifiers.send(name, "delete", imdb.logger)
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
	for name := range imdb.imageMap {
		names = append(names, name)
	}
	return names
}

func (imdb *ImageDataBase) countImages() uint {
	imdb.RLock()
	defer imdb.RUnlock()
	return uint(len(imdb.imageMap))
}

func (imdb *ImageDataBase) registerAddNotifier() <-chan string {
	channel := make(chan string, 1)
	imdb.Lock()
	defer imdb.Unlock()
	imdb.addNotifiers[channel] = channel
	return channel
}

func (imdb *ImageDataBase) unregisterAddNotifier(channel <-chan string) {
	imdb.Lock()
	defer imdb.Unlock()
	delete(imdb.addNotifiers, channel)
}

func (imdb *ImageDataBase) registerDeleteNotifier() <-chan string {
	channel := make(chan string, 1)
	imdb.Lock()
	defer imdb.Unlock()
	imdb.deleteNotifiers[channel] = channel
	return channel
}

func (imdb *ImageDataBase) unregisterDeleteNotifier(channel <-chan string) {
	imdb.Lock()
	defer imdb.Unlock()
	delete(imdb.deleteNotifiers, channel)
}

func (n notifiers) send(name string, operation string, logger *log.Logger) {
	if len(n) < 1 {
		return
	} else {
		plural := "s"
		if len(n) < 2 {
			plural = ""
		}
		logger.Printf("Sending %s notification to: %d listener%s\n",
			operation, len(n), plural)
	}
	for _, sendChannel := range n {
		go func(channel chan<- string) {
			channel <- name
		}(sendChannel)
	}
}
