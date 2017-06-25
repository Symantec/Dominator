package nulllogger

type Logger struct{}

func New() Logger {
	return Logger{}
}

func (Logger) Fatal(v ...interface{})                              {}
func (Logger) Fatalf(format string, v ...interface{})              {}
func (Logger) Fatalln(v ...interface{})                            {}
func (Logger) Panic(v ...interface{})                              {}
func (Logger) Panicf(format string, v ...interface{})              {}
func (Logger) Panicln(v ...interface{})                            {}
func (Logger) Print(v ...interface{})                              {}
func (Logger) Printf(format string, v ...interface{})              {}
func (Logger) Println(v ...interface{})                            {}
func (Logger) Debug(level uint8, v ...interface{})                 {}
func (Logger) Debugf(level uint8, format string, v ...interface{}) {}
func (Logger) Debugln(level uint8, v ...interface{})               {}
