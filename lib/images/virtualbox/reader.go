package virtualbox

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/format"
)

type headerType struct {
	Header          [0x40]byte
	Signature       uint32
	VersionMajor    uint16
	VersionMinor    uint16
	HeaderSize      uint32
	ImageType       uint32
	ImageFlags      uint32
	Description     [0x100]byte
	OffsetBocks     uint32
	OffsetData      uint32
	Cylinders       uint32
	Heads           uint32
	Sectors         uint32
	SectorSize      uint32
	Unused          [4]byte
	DiscSize        uint64
	BlockSize       uint32
	BlockExtraData  uint32
	BlocksInHDD     uint32
	BlocksAllocated uint32
	UuidImage       uint64
	UuidLastSnap    uint64
	UuidLink        uint64
	UuidParent      uint64
}

func newReader(rawReader io.Reader) (*Reader, error) {
	r := bufio.NewReaderSize(rawReader, 1<<20)
	var header headerType
	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("error reading VDI header: %s", err)
	}
	if header.Signature != 0xbeda107f {
		return nil, fmt.Errorf("%x not a VDI signature", header.Signature)
	}
	if header.VersionMajor < 1 {
		return nil,
			fmt.Errorf("VDI major version: %d not supported",
				header.VersionMajor)
	}
	if header.ImageType != 1 {
		return nil, fmt.Errorf("VDI image type: %d not supported",
			header.ImageType)
	}
	if header.BlockSize != 1<<20 {
		return nil, fmt.Errorf("VDI block size: 0x%x (%s) not supported",
			header.BlockSize, format.FormatBytes(uint64(header.BlockSize)))
	}
	mul := uint64(header.BlockSize) * uint64(header.BlocksInHDD)
	if mul != header.DiscSize {
		return nil, fmt.Errorf("blockSize*blocksInHdd: %d != discSize: %d",
			mul, header.DiscSize)
	}
	r.Reset(rawReader) // Discard until 1 MiB boundary.
	lastPointer := int32(-1)
	blockMap := make(map[uint32]struct{}, header.BlocksInHDD)
	for index := uint32(0); index < header.BlocksInHDD; index++ {
		var pointer int32
		if err := binary.Read(r, binary.LittleEndian, &pointer); err != nil {
			return nil,
				fmt.Errorf("error reading block pointer: %d: %s", index, err)
		}
		if pointer >= 0 {
			if pointer <= lastPointer {
				return nil,
					fmt.Errorf("VDI pointer: %d not greater than last: %d",
						pointer, lastPointer)
			}
			blockMap[index] = struct{}{}
		}
		lastPointer = pointer
	}
	r.Reset(rawReader) // Discard until 1 MiB boundary.
	return &Reader{
		Description:  string(header.Description[:]),
		Header:       string(header.Header[:]),
		MajorVersion: header.VersionMajor,
		MinorVersion: header.VersionMinor,
		Size:         header.DiscSize,
		blockMap:     blockMap,
		blockSize:    header.BlockSize,
		blocksInHDD:  header.BlocksInHDD,
		reader:       r,
	}, nil
}

func (r *Reader) read(p []byte) (int, error) {
	if r.blockIndex >= r.blocksInHDD {
		return 0, io.EOF
	}
	blockIndex := r.blockIndex
	if maxLength := r.blockSize - r.blockOffset; uint32(len(p)) >= maxLength {
		p = p[:maxLength]
		r.blockIndex++
		r.blockOffset = 0
	} else {
		r.blockOffset += uint32(len(p))
	}
	if _, ok := r.blockMap[blockIndex]; !ok {
		for index := range p {
			p[index] = 0
		}
		return len(p), nil
	}
	return io.ReadFull(r.reader, p)
}
