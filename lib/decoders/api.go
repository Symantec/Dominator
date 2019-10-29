package decoders

import (
	"io"
)

type Decoder interface {
	Decode(value interface{}) error
}

type DecoderGenerator func(r io.Reader) Decoder

func RegisterDecoder(extension string, decoderGenerator DecoderGenerator) {
	registerDecoder(extension, decoderGenerator)
}

func DecodeFile(filename string, value interface{}) error {
	return decodeFile(filename, value)
}
