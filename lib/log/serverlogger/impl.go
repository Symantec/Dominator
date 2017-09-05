package serverlogger

import (
	"errors"
	"github.com/Symantec/Dominator/lib/log/debuglogger"
	"github.com/Symantec/Dominator/lib/logbuf"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/logger"
	"log"
	"strings"
	"sync"
)

type loggerMapT struct {
	sync.Mutex
	loggerMap map[string]*Logger
}

var loggerMap *loggerMapT = &loggerMapT{loggerMap: make(map[string]*Logger)}

func init() {
	srpc.RegisterName("Logger", loggerMap)
}

func newLogger(name string) *Logger {
	loggerMap.Lock()
	defer loggerMap.Unlock()
	if _, ok := loggerMap.loggerMap[name]; ok {
		panic("logger already exists: " + name)
	}
	circularBuffer := logbuf.New()
	debugLogger := debuglogger.New(log.New(circularBuffer, "", log.LstdFlags))
	if *initialLogDebugLevel >= 0 {
		debugLogger.SetLevel(int16(*initialLogDebugLevel))
	}
	logger := &Logger{
		Logger:         debugLogger,
		circularBuffer: circularBuffer,
	}
	loggerMap.loggerMap[name] = logger
	return logger
}

func (t *loggerMapT) Debug(conn *srpc.Conn,
	request proto.DebugRequest,
	reply *proto.DebugResponse) error {
	loggerMap.Lock()
	defer loggerMap.Unlock()
	if logger, ok := loggerMap.loggerMap[request.Name]; !ok {
		return errors.New("unknown logger: " + request.Name)
	} else {
		logger.Debugf(request.Level, "Logger.Debug(%d): %s\n",
			request.Level, strings.Join(request.Args, " "))
		return nil
	}
}

func (t *loggerMapT) Print(conn *srpc.Conn,
	request proto.PrintRequest,
	reply *proto.PrintResponse) error {
	loggerMap.Lock()
	defer loggerMap.Unlock()
	if logger, ok := loggerMap.loggerMap[request.Name]; !ok {
		return errors.New("unknown logger: " + request.Name)
	} else {
		logger.Println("Logger.Print():", strings.Join(request.Args, " "))
		return nil
	}
}

func (t *loggerMapT) SetDebugLevel(conn *srpc.Conn,
	request proto.SetDebugLevelRequest,
	reply *proto.SetDebugLevelResponse) error {
	loggerMap.Lock()
	defer loggerMap.Unlock()
	if logger, ok := loggerMap.loggerMap[request.Name]; !ok {
		return errors.New("unknown logger: " + request.Name)
	} else {
		logger.Printf("Logger.SetDebugLevel(%d)\n", request.Level)
		logger.SetLevel(request.Level)
		return nil
	}
}
