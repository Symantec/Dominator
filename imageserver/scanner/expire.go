package scanner

import (
	"os"
	"path"
	"time"

	"github.com/Symantec/Dominator/lib/image"
)

func imageIsExpired(image *image.Image) bool {
	if !image.ExpiresAt.IsZero() && image.ExpiresAt.Sub(time.Now()) <= 0 {
		return true
	}
	return false
}

func (imdb *ImageDataBase) scheduleExpiration(image *image.Image,
	name string) bool {
	if image.ExpiresAt.IsZero() {
		return false
	}
	duration := image.ExpiresAt.Sub(time.Now())
	if duration <= 0 {
		return true
	}
	time.AfterFunc(duration, func() {
		imdb.logger.Printf("Auto expiring (deleting) image: %s\n", name)
		if err := os.Remove(path.Join(imdb.baseDir, name)); err != nil {
			imdb.logger.Println(err)
		}
		imdb.Lock()
		defer imdb.Unlock()
		imdb.deleteImageAndUpdateUnreferencedObjectsList(name)
	})
	return false
}
