package image

func (annotation *Annotation) replaceStrings(replaceFunc func(string) string) {
	if annotation != nil {
		annotation.URL = replaceFunc(annotation.URL)
	}
}

func (image *Image) replaceStrings(replaceFunc func(string) string) {
	image.CreatedBy = replaceFunc(image.CreatedBy)
	image.Filter.ReplaceStrings(replaceFunc)
	image.FileSystem.ReplaceStrings(replaceFunc)
	image.Triggers.ReplaceStrings(replaceFunc)
	image.ReleaseNotes.replaceStrings(replaceFunc)
	image.BuildLog.replaceStrings(replaceFunc)
	for index := range image.Packages {
		pkg := &image.Packages[index]
		pkg.replaceStrings(replaceFunc)
	}
}

func (pkg *Package) replaceStrings(replaceFunc func(string) string) {
	pkg.Version = replaceFunc(pkg.Version)
}
