package logger

type DebugRequest struct {
	Args  []string
	Name  string
	Level uint8
}

type DebugResponse struct{}

type PrintRequest struct {
	Args []string
	Name string
}

type PrintResponse struct{}

type SetDebugLevelRequest struct {
	Name  string
	Level int16
}

type SetDebugLevelResponse struct{}

type WatchRequest struct {
	DebugLevel   int16
	DumpBuffer   bool
	ExcludeRegex string // Empty: nothing excluded. Processed after includes.
	IncludeRegex string // Empty: everything included.
	Name         string
}

type WatchResponse struct {
	Error string
} // Log data are streamed afterwards.
