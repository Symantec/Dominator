package serverlogger

import (
	"github.com/Symantec/Dominator/lib/log/debuglogger"
	"github.com/Symantec/Dominator/lib/logbuf"
	"log"
)

func newLogger() *Logger {
	circularBuffer := logbuf.New()
	logger := debuglogger.New(log.New(circularBuffer, "", log.LstdFlags))
	if *initialLogDebugLevel >= 0 {
		logger.SetLevel(int16(*initialLogDebugLevel))
	}
	return &Logger{
		Logger:         logger,
		circularBuffer: circularBuffer,
	}
}
