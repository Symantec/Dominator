package filesystem

func (fs *FileSystem) estimateUsage(blockSize uint64) uint64 {
	if blockSize < 1 {
		blockSize = 4096
	}
	fileOrder := sizeToOrder(blockSize)
	var totalDataBytes uint64
	for _, inode := range fs.InodeTable {
		switch inode := inode.(type) {
		case *DirectoryInode:
			totalDataBytes += blockSize
		case *RegularInode:
			// Round up to the nearest page size.
			size := (inode.Size >> fileOrder) << fileOrder
			if size < inode.Size {
				size += 1 << fileOrder
			}
			totalDataBytes += size
		case *SymlinkInode:
			totalDataBytes += 64
		}
	}
	return totalDataBytes
}

func sizeToOrder(blockSize uint64) uint {
	order := uint(0)
	for i := uint(0); i < 64; i++ {
		if 1<<i&blockSize != 0 {
			order = i
		}
	}
	return order
}
