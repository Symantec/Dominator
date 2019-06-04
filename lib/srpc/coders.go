package srpc

import (
	"encoding/gob"
	"encoding/json"
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

type jsonCoder struct{}

func (c *jsonCoder) MakeDecoder(r io.Reader) Decoder {
	return json.NewDecoder(r)
}

func (c *jsonCoder) MakeEncoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}
