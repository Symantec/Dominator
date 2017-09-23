package format

import (
	"fmt"
	"time"
)

// Duration is similar to the time.Duration.String method from the standard
// library but is more readable and shows only 3 digits of precision when
// duration is less than 1 minute.
func Duration(duration time.Duration) string {
	if ns := duration.Nanoseconds(); ns < 1000 {
		return fmt.Sprintf("%dns", ns)
	} else if us := float64(duration) / float64(time.Microsecond); us < 1000 {
		return fmt.Sprintf("%.3gµs", us)
	} else if ms := float64(duration) / float64(time.Millisecond); ms < 1000 {
		return fmt.Sprintf("%.3gms", ms)
	} else if s := float64(duration) / float64(time.Second); s < 60 {
		return fmt.Sprintf("%.3gs", s)
	} else {
		duration -= duration % time.Second
		day := time.Hour * 24
		if duration < day {
			return duration.String()
		}
		days := duration / day
		duration %= day
		return fmt.Sprintf("%dd%s", days, duration)
	}
}

// FormatBytes returns a string with the number of bytes specified converted
// into a human-friendly format with a binary multiplier (i.e. GiB).
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
