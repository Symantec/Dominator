package scanner

import (
	"github.com/Symantec/Dominator/lib/image"
	"os"
	"path"
	"time"
)

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
		delete(imdb.imageMap, name)
	})
	return false
}
