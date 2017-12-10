package logutil

import "github.com/Symantec/Dominator/lib/log"

func LogMemory(logger log.DebugLogger, level int16, message string) {
	logMemory(logger, level, message)
}
