package scanner

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"io"
	"sync"
)

// TODO: the types should probably be moved into a separate package, leaving
//       behind the scanner code.

type Object struct {
	length uint64
}

type ImageDataBase struct {
	imageMap  map[string]*image.Image
	objectMap map[hash.Hash]*Object
	sync.RWMutex
}

func LoadImageDataBase(baseDir string) (*ImageDataBase, error) {
	return loadImageDataBase(baseDir)
}

func (imdb *ImageDataBase) WriteHtml(writer io.Writer) {
	imdb.writeHtml(writer)
}

func (imdb *ImageDataBase) AddImage(image *image.Image, name string) error {
	return imdb.addImage(image, name)
}

func (imdb *ImageDataBase) CheckImage(name string) bool {
	return imdb.checkImage(name)
}

func (imdb *ImageDataBase) DeleteImage(name string) error {
	return imdb.deleteImage(name)
}

func (imdb *ImageDataBase) GetImage(name string) *image.Image {
	return imdb.getImage(name)
}

func (imdb *ImageDataBase) ListImages() []string {
	return imdb.listImages()
}
