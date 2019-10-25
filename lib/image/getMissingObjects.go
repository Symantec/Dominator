package image

import (
	"errors"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
)

func (image *Image) getMissingObjects(objectServer objectserver.ObjectServer,
	objectsGetter objectserver.ObjectsGetter, logger log.DebugLogger) error {
	missingObjects, err := image.ListMissingObjects(objectServer)
	if err != nil {
		return err
	}
	if len(missingObjects) < 1 {
		return nil
	}
	var numObjects uint64
	image.ForEachObject(func(hashVal hash.Hash) error {
		numObjects++
		return nil
	})
	logger.Printf("downloading %d of %d objects\n",
		len(missingObjects), numObjects)
	startTime := time.Now()
	objectsReader, err := objectsGetter.GetObjects(missingObjects)
	if err != nil {
		return errors.New("error downloading objects: " + err.Error())
	}
	defer objectsReader.Close()
	var totalBytes uint64
	for _, hash := range missingObjects {
		length, reader, err := objectsReader.NextObject()
		if err != nil {
			return err
		}
		_, _, err = objectServer.AddObject(reader, length, &hash)
		reader.Close()
		if err != nil {
			return err
		}
		totalBytes += length
	}
	timeTaken := time.Since(startTime)
	logger.Printf("downloaded %d objects, %s in %s (%s/s)\n",
		len(missingObjects), format.FormatBytes(totalBytes), timeTaken,
		format.FormatBytes(uint64(float64(totalBytes)/timeTaken.Seconds())))
	return nil
}
