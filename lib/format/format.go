package format

import (
	"fmt"
)

func FormatBytes(bytes uint64) string {
	if bytes>>30 > 100 {
		return fmt.Sprintf("%d GiB", bytes>>30)
	} else if bytes>>20 > 100 {
		return fmt.Sprintf("%d MiB", bytes>>20)
	} else if bytes>>10 > 100 {
		return fmt.Sprintf("%d KiB", bytes>>10)
	} else {
		return fmt.Sprintf("%d B", bytes)
	}
}
