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
	ImageName       string
	Filter          [][]string
	DataSize        uint64
	CompressionType uint
}

type AddResponse StatusResponse

type CheckRequest struct {
	ImageName string
}

type CheckResponse struct {
	ImageExists bool
}

type DeleteRequest struct {
	ImageName string
}

type DeleteResponse StatusResponse

type ListRequest struct {
}

type ListResponse struct {
	ImageNames []string
}
