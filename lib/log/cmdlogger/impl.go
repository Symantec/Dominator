package cmdlogger

import (
	"github.com/Symantec/Dominator/lib/log/debuglogger"
	"log"
	"os"
)

func newLogger() *debuglogger.Logger {
	logFlags := 0
	if *logDatestamps {
		logFlags |= log.LstdFlags
	}
	logger := debuglogger.New(log.New(os.Stderr, "", logFlags))
	logger.SetLevel(int16(*logDebugLevel))
	return logger
}
