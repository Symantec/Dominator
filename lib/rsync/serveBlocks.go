package rsync

import (
	"crypto/sha512"
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	proto "github.com/Cloud-Foundations/Dominator/proto/rsync"
)

func serveBlocks(conn Conn, decoder Decoder, encoder Encoder,
	reader io.ReadSeeker, length uint64) error {
	var request proto.GetBlocksRequest
	if err := decoder.Decode(&request); err != nil {
		return err
	}
	if request.BlockOrder < 9 || request.BlockOrder > 32 {
		return encoder.Encode(proto.Block{Error: "bad block order"})
	}
	blockSize := int64(1) << request.BlockOrder
	var index uint64
	for ; index < request.NumBlocks; index++ {
		hasher := sha512.New()
		if _, err := io.CopyN(hasher, reader, blockSize); err != nil {
			return err
		}
		var localHash, remoteHash hash.Hash
		copy(localHash[:], hasher.Sum(nil))
		if nRead, err := conn.Read(remoteHash[:]); err != nil {
			return encoder.Encode(proto.Block{Error: err.Error()})
		} else if nRead != len(remoteHash) {
			return encoder.Encode(proto.Block{Error: "short read"})
		}
		if remoteHash != localHash {
			if _, err := reader.Seek(-blockSize, io.SeekCurrent); err != nil {
				return encoder.Encode(proto.Block{Error: err.Error()})
			}
			block := proto.Block{Index: index, Size: uint64(blockSize)}
			if err := encoder.Encode(block); err != nil {
				return err
			}
			if _, err := io.CopyN(conn, reader, blockSize); err != nil {
				return encoder.Encode(proto.Block{Error: err.Error()})
			}
		}
	}
	block := proto.Block{
		Index: index,
		Size:  length - index<<request.BlockOrder,
	}
	if err := encoder.Encode(block); err != nil {
		return err
	}
	if block.Size < 1 {
		return nil
	}
	if _, err := io.CopyN(conn, reader, int64(block.Size)); err != nil {
		return err
	}
	return encoder.Encode(proto.Block{})
}
