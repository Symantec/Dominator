package srpc

import (
	"encoding/gob"
	"io"
)

type coderMaker interface {
	MakeDecoder(r io.Reader) Decoder
	MakeEncoder(w io.Writer) Encoder
}

type gobCoder struct{}

func (c *gobCoder) MakeDecoder(r io.Reader) Decoder {
	return gob.NewDecoder(r)
}

func (c *gobCoder) MakeEncoder(w io.Writer) Encoder {
	return gob.NewEncoder(w)
}
