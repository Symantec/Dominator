package imageserver

type StatusResponse struct {
	Success     bool
	ErrorString string
}

const (
	UNCOMPRESSED = iota
	GZIP
)

type AddRequest struct {
	Name            string
	Filter          [][]string
	DataSize        uint64
	CompressionType uint
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
