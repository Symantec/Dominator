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
