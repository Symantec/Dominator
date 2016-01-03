package filesystem

import (
	"encoding/gob"
	"io"
)

type encodedFileSystemType struct {
	FileSystem
	InodeTableLength uint64
}

type numberedInode struct {
	InodeNumber uint64
	GenericInode
}

func (fs *FileSystem) encode(writer io.Writer) error {
	// Make a copy of the FileSystem so that the InodeTable can be ripped out
	// and streamed afterwards.
	var encodedFileSystem encodedFileSystemType
	inodeTable := fs.InodeTable
	encodedFileSystem.FileSystem = *fs
	encodedFileSystem.FileSystem.InodeTable = nil
	encodedFileSystem.InodeTableLength = uint64(len(inodeTable))
	encoder := gob.NewEncoder(writer)
	if err := encoder.Encode(encodedFileSystem); err != nil {
		return err
	}
	// Stream out the InodeTable.
	for inodeNumber, genericInode := range inodeTable {
		var inode numberedInode
		inode.InodeNumber = inodeNumber
		inode.GenericInode = genericInode
		if err := encoder.Encode(inode); err != nil {
			return err
		}
	}
	return nil
}

func decode(reader io.Reader) (*FileSystem, error) {
	decoder := gob.NewDecoder(reader)
	fs := new(FileSystem)
	var decodedFileSystem encodedFileSystemType
	if err := decoder.Decode(&decodedFileSystem); err != nil {
		return nil, err
	}
	*fs = decodedFileSystem.FileSystem
	// Stream in the InodeTable.
	numInodes := decodedFileSystem.InodeTableLength
	fs.InodeTable = make(InodeTable, numInodes)
	for index := uint64(0); index < numInodes; index++ {
		var inode numberedInode
		if err := decoder.Decode(&inode); err != nil {
			return nil, err
		}
		fs.InodeTable[inode.InodeNumber] = inode.GenericInode
	}
	return fs, nil
}
