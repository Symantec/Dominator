package scanner

import (
	"io"
)

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
}

func LoadImageDataBase(baseDir string) (*ImageDataBase, error) {
	return loadImageDataBase(baseDir)
}

func (imdb *ImageDataBase) WriteHtml(writer io.Writer) {
	//imdb.writeHtml(writer)
}
