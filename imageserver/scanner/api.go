package scanner

import (
	"io"
	"sync"
)

// TODO: the types should probably be moved into a separate package, leaving
//       behind the scanner code.

type FilterEntry string

type Filter []FilterEntry

type Directory struct {
}

type Image struct {
	filter       Filter
	topDirectory *Directory
}

type Hash [64]byte

type Object struct {
	length uint64
}

type ImageDataBase struct {
	imageMap  map[string]*Image
	objectMap map[Hash]*Object
	sync.RWMutex
}

func LoadImageDataBase(baseDir string) (*ImageDataBase, error) {
	return loadImageDataBase(baseDir)
}

func (imdb *ImageDataBase) WriteHtml(writer io.Writer) {
	imdb.writeHtml(writer)
}

func (imdb *ImageDataBase) AddImage(image *Image, name string) error {
	return imdb.addImage(image, name)
}

func (imdb *ImageDataBase) CheckImage(name string) bool {
	return imdb.checkImage(name)
}

func (imdb *ImageDataBase) DeleteImage(name string) error {
	return imdb.deleteImage(name)
}

func (imdb *ImageDataBase) GetImage(name string) *Image {
	return imdb.getImage(name)
}

func (imdb *ImageDataBase) ListImages() []string {
	return imdb.listImages()
}
