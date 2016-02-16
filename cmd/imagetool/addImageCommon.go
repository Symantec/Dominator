package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/lib/triggers"
	"os"
)

func loadImageFiles(image *image.Image, objectClient *objectclient.ObjectClient,
	filterFilename, triggersFilename string) error {
	var err error
	if filterFilename != "" {
		image.Filter, err = filter.LoadFilter(filterFilename)
		if err != nil {
			return err
		}
	}
	if err := loadTriggers(image, triggersFilename); err != nil {
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

func loadTriggers(image *image.Image, triggersFilename string) error {
	triggersFile, err := os.Open(triggersFilename)
	if err != nil {
		return err
	}
	defer triggersFile.Close()
	decoder := json.NewDecoder(triggersFile)
	var trig triggers.Triggers
	if err = decoder.Decode(&trig.Triggers); err != nil {
		return errors.New("error decoding triggers " + err.Error())
	}
	image.Triggers = &trig
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
