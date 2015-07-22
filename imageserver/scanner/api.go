package scanner

type ImageDataBase struct {
}

func LoadImageDataBase(baseDir string) (*ImageDataBase, error) {
	return loadImageDataBase(baseDir)
}
