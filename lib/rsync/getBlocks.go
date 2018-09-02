package rsync

import (
	"crypto/sha512"
	"fmt"
	"io"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/hash"
	proto "github.com/Symantec/Dominator/proto/rsync"
)

type measuringConn struct {
	Conn
	stats Stats
}

func getBlocks(rawConn Conn, decoder Decoder, encoder Encoder, reader io.Reader,
	writer io.WriteSeeker, totalBytes, readerBytes uint64) (Stats, error) {
	blockOrder := sizeToOrder(totalBytes) >> 1
	if blockOrder < 9 {
		blockOrder = 9
	} else if blockOrder > 32 {
		blockOrder = 32
	}
	blockSize := uint64(1 << blockOrder)
	if reader == nil {
		readerBytes = 0
	}
	numBlocks := readerBytes >> blockOrder
	request := proto.GetBlocksRequest{
		BlockOrder: blockOrder,
		NumBlocks:  numBlocks,
	}
	conn := &measuringConn{Conn: rawConn}
	if err := encoder.Encode(request); err != nil {
		return Stats{}, fmt.Errorf("error encoding request: %s", err)
	}
	if err := conn.Flush(); err != nil {
		return Stats{}, err
	}
	errChannel := make(chan error, 1)
	go func() { errChannel <- readBlocks(writer, decoder, conn, blockOrder) }()
	for index := uint64(0); index < numBlocks; index++ {
		select {
		case err := <-errChannel:
			if err != nil {
				return Stats{}, err
			}
			return Stats{}, errors.New("premature end of blocks")
		default:
		}
		hasher := sha512.New()
		if _, err := io.CopyN(hasher, reader, int64(blockSize)); err != nil {
			return Stats{}, err
		}
		var hashVal hash.Hash
		copy(hashVal[:], hasher.Sum(nil))
		if _, err := conn.Write(hashVal[:]); err != nil {
			return Stats{}, err
		}
		if index == 0 {
			if err := conn.Flush(); err != nil {
				return Stats{}, err
			}
		}
	}
	if err := conn.Flush(); err != nil {
		return Stats{}, err
	}
	if err := <-errChannel; err != nil {
		return Stats{}, err
	}
	return conn.stats, nil
}

func readBlocks(writer io.WriteSeeker, decoder Decoder, reader io.Reader,
	blockOrder uint8) error {
	var numBytesReceived uint64
	for {
		var block proto.Block
		if err := decoder.Decode(&block); err != nil {
			return fmt.Errorf("error decoding block: %s", err)
		}
		if err := errors.New(block.Error); err != nil {
			return err
		}
		if block.Size < 1 {
			return nil
		}
		offset := int64(block.Index << blockOrder)
		if _, err := writer.Seek(offset, io.SeekStart); err != nil {
			return err
		}
		if _, err := io.CopyN(writer, reader, int64(block.Size)); err != nil {
			return err
		}
		numBytesReceived += block.Size
	}
}

func sizeToOrder(blockSize uint64) uint8 {
	order := uint8(0)
	for i := uint8(0); i < 64; i++ {
		if 1<<i&blockSize != 0 {
			order = i
		}
	}
	return order
}

func (conn *measuringConn) Read(b []byte) (int, error) {
	nRead, err := conn.Conn.Read(b)
	conn.stats.NumRead += uint64(nRead)
	return nRead, err
}

func (conn *measuringConn) Write(b []byte) (int, error) {
	nWritten, err := conn.Conn.Write(b)
	conn.stats.NumWritten += uint64(nWritten)
	return nWritten, err
}
