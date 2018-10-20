package fsutil

import (
	"bufio"
	"bytes"
	"crypto/sha512"
	"fmt"
	"io"
)

func newChecksumReader(reader io.Reader) *ChecksumReader {
	r := new(ChecksumReader)
	r.checksummer = sha512.New()
	if _, ok := reader.(io.ByteReader); !ok {
		r.reader = bufio.NewReader(reader)
	} else {
		r.reader = reader
	}
	return r
}

func (r *ChecksumReader) getChecksum() []byte {
	return r.checksummer.Sum(nil)
}

func (r *ChecksumReader) read(p []byte) (int, error) {
	if nRead, err := r.reader.Read(p); err != nil && err != io.EOF {
		return nRead, err
	} else if _, e := r.checksummer.Write(p[:nRead]); e != nil {
		return nRead, e
	} else {
		return nRead, err
	}
}

func (r *ChecksumReader) readByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := r.read(buf)
	return buf[0], err
}

func (r *ChecksumReader) verifyChecksum() error {
	buf := make([]byte, r.checksummer.Size())
	nRead, err := io.ReadAtLeast(r.reader, buf, len(buf))
	if err != nil {
		return err
	}
	if nRead != r.checksummer.Size() {
		return fmt.Errorf(
			"ChecksumReader.Checksum(): expected: %d got: %d bytes",
			r.checksummer.Size(), nRead)
	}
	if !bytes.Equal(buf, r.getChecksum()) {
		return ErrorChecksumMismatch
	}
	return nil
}

func newChecksumWriter(writer io.Writer) *ChecksumWriter {
	w := new(ChecksumWriter)
	w.checksummer = sha512.New()
	w.writer = writer
	return w
}

func (w *ChecksumWriter) write(p []byte) (int, error) {
	if _, err := w.checksummer.Write(p); err != nil {
		return 0, err
	}
	return w.writer.Write(p)
}

func (w *ChecksumWriter) writeChecksum() error {
	_, err := w.writer.Write(w.checksummer.Sum(nil))
	return err
}
