package cmdlogger

import (
	"flag"
	"github.com/Symantec/Dominator/lib/log/debuglogger"
)

var (
	logDatestamps = flag.Bool("logDatestamps", false,
		"If true, prefix logs with datestamps")
	logDebugLevel = flag.Int("logDebugLevel", -1, "Debug log level")
)

// New will create a debuglogger.Logger which writes to the standard error.
// The following command-line flags are registered:
//   -logDatestamps: if true, prefix logs with datestamps
//   -logDebugLevel: debug log level
func New() *debuglogger.Logger {
	return newLogger()
}

// SetDatestampsDefault will change the default for the -logDatestamps command
// line flag. This should be called before flag.Parse().
func SetDatestampsDefault(defaultValue bool) {
	*logDatestamps = defaultValue
}
