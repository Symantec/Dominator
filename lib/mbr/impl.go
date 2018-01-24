package mbr

import (
	"fmt"
	"os"
	"os/exec"
)

func decode(file *os.File) (*Mbr, error) {
	var mbr Mbr
	if _, err := file.ReadAt(mbr.raw[:], 0); err != nil {
		return nil, err
	}
	if mbr.raw[0x1FE] == 0x55 && mbr.raw[0x1FF] == 0xAA {
		return &mbr, nil
	}
	return nil, nil
}

func (mbr *Mbr) getPartitionOffset(index uint) uint64 {
	partitionOffset := 0x1BE + 0x10*index
	return 512 * (uint64(mbr.raw[partitionOffset+8]) +
		uint64(mbr.raw[partitionOffset+9])<<8 +
		uint64(mbr.raw[partitionOffset+10])<<16 +
		uint64(mbr.raw[partitionOffset+11])<<24)
}

func (mbr *Mbr) getPartitionSize(index uint) uint64 {
	partitionOffset := 0x1BE + 0x10*index
	return 512 * (uint64(mbr.raw[partitionOffset+12]) +
		uint64(mbr.raw[partitionOffset+13])<<8 +
		uint64(mbr.raw[partitionOffset+14])<<16 +
		uint64(mbr.raw[partitionOffset+15])<<24)
}

func writeDefault(filename string) error {
	cmd := exec.Command("parted", "-s", "-a", "optimal", filename,
		"mklabel", "msdos", "mkpart", "primary", "ext2", "0%", "100%")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error partitioning: %s: %s: %s",
			filename, err, output)
	}
	return nil
}
