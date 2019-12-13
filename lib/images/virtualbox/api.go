package virtualbox

import (
	"io"
)

type Reader struct {
	Description  string
	Header       string
	MajorVersion uint16
	MinorVersion uint16
	Size         uint64
	blockIndex   uint32
	blockMap     map[uint32]struct{}
	blockOffset  uint32
	blockSize    uint32
	blocksInHDD  uint32
	reader       io.Reader
}

func NewReader(r io.Reader) (*Reader, error) {
	return newReader(r)
}

func (r *Reader) Read(p []byte) (int, error) {
	return r.read(p)
}
