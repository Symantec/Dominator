package logutil

import "github.com/Cloud-Foundations/Dominator/lib/log"

func LogMemory(logger log.DebugLogger, level int16, message string) {
	logMemory(logger, level, message)
}
