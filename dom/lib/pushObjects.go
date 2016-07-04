package lib

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectserver"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"log"
)

func (sub *Sub) pushObjects(objectsToPush map[hash.Hash]struct{},
	objectGetter objectserver.ObjectGetter, logger *log.Logger) error {
	objQ, err := objectclient.NewObjectAdderQueue(sub.Client)
	if err != nil {
		logger.Printf("Error creating object adder queue for: %s: %s\n",
			sub, err)
		return err
	}
	for hashVal := range objectsToPush {
		length, reader, err := objectGetter.GetObject(hashVal)
		if err != nil {
			logger.Printf("Error getting object: %x: %s\n", hashVal, err)
			objQ.Close()
			return ErrorFailedToGetObject
		}
		_, err = objQ.Add(reader, length)
		reader.Close()
		if err != nil {
			logger.Printf("Error pushing: %x to: %s: %s\n", hashVal, sub, err)
			objQ.Close()
			return err
		}
	}
	if err := objQ.Close(); err != nil {
		logger.Printf("Error pushing objects to: %s: %s\n", sub, err)
		return err
	}
	return nil
}
