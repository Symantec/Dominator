package cmdlogger

import (
	"flag"
	"log"

	"github.com/Symantec/Dominator/lib/log/debuglogger"
)

func init() {
	flag.BoolVar(&stdOptions.Datestamps, "logDatestamps", false,
		"If true, prefix logs with datestamps")
	flag.IntVar(&stdOptions.DebugLevel, "logDebugLevel", -1, "Debug log level")
	flag.BoolVar(&stdOptions.Subseconds, "logSubseconds", false,
		"If true, datestamps will have subsecond resolution")
}

func newLogger(options Options) *debuglogger.Logger {
	if options.DebugLevel < -1 {
		options.DebugLevel = -1
	}
	if options.DebugLevel > 65535 {
		options.DebugLevel = 65535
	}
	logFlags := 0
	if options.Datestamps {
		logFlags |= log.LstdFlags
		if options.Subseconds {
			logFlags |= log.Lmicroseconds
		}
	}
	logger := debuglogger.New(log.New(options.Writer, "", logFlags))
	logger.SetLevel(int16(options.DebugLevel))
	return logger
}
