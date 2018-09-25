package rsync

import (
	"io"
)

type Conn interface {
	Flush() error
	io.Reader
	io.Writer
}

type Decoder interface {
	Decode(e interface{}) error
}

type Encoder interface {
	Encode(e interface{}) error
}

type Stats struct {
	NumRead    uint64
	NumWritten uint64
}

func GetBlocks(conn Conn, decoder Decoder, encoder Encoder, reader io.Reader,
	writer io.WriteSeeker, totalBytes, readerBytes uint64) (Stats, error) {
	return getBlocks(conn, decoder, encoder, reader, writer, totalBytes,
		readerBytes)
}

func ServeBlocks(conn Conn, decoder Decoder, encoder Encoder,
	reader io.ReadSeeker, length uint64) error {
	return serveBlocks(conn, decoder, encoder, reader, length)
}
