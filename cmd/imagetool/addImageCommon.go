package main

import (
	"bufio"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/image"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/triggers"
	"os"
)

func loadImageFiles(image *image.Image, objectClient *objectclient.ObjectClient,
	filterFilename, triggersFilename string) error {
	var err error
	if filterFilename != "" {
		image.Filter, err = filter.Load(filterFilename)
		if err != nil {
			return err
		}
	}
	image.Triggers, err = triggers.Load(triggersFilename)
	if err != nil {
		return err
	}
	image.BuildLog, err = getAnnotation(objectClient, *buildLog)
	if err != nil {
		return err
	}
	image.ReleaseNotes, err = getAnnotation(objectClient, *releaseNotes)
	if err != nil {
		return err
	}
	return nil
}

func getAnnotation(objectClient *objectclient.ObjectClient, name string) (
	*image.Annotation, error) {
	if name == "" {
		return nil, nil
	}
	file, err := os.Open(name)
	if err != nil {
		return &image.Annotation{URL: name}, nil
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(file)
	hash, _, err := objectClient.AddObject(reader, uint64(fi.Size()), nil)
	return &image.Annotation{Object: &hash}, nil
}
