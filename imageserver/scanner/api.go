package scanner

type FilterEntry string

type Filter []FilterEntry

type Directory struct {
}

type Image struct {
	filter       Filter
	topDirectory *Directory
}

type Hash [64]byte

type ImageDataBase struct {
	imageMap map[string]*Image
}

func LoadImageDataBase(baseDir string) (*ImageDataBase, error) {
	return loadImageDataBase(baseDir)
}
