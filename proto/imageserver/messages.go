package imageserver

type StatusResponse struct {
	Success     bool
	ErrorString string
}

type AddRequest struct {
	Name   string
	Filter [][]string
}

type AddResponse StatusResponse

type CheckRequest struct {
	Name string
}

type CheckResponse struct {
	ImageExists bool
}

type DeleteRequest struct {
	Name string
}

type DeleteResponse StatusResponse

type ListRequest struct {
}

type ListResponse struct {
	ImageNames [][]string
}
