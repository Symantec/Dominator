package mbr

import (
	"os"
)

type Mbr struct {
	raw [512]byte
}

func Decode(file *os.File) (*Mbr, error) {
	return decode(file)
}

func (mbr *Mbr) GetNumPartitions() uint {
	return 4
}

func (mbr *Mbr) GetPartitionOffset(index uint) uint64 {
	return mbr.getPartitionOffset(index)
}

func (mbr *Mbr) GetPartitionSize(index uint) uint64 {
	return mbr.getPartitionSize(index)
}
