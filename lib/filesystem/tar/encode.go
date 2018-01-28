package tar

import (
	"archive/tar"
	"fmt"
	"io"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectserver"
)

func encode(tarWriter *tar.Writer, fileSystem *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter) error {
	hashList := getOrderedObjectsList(fileSystem)
	objectsReader, err := objectsGetter.GetObjects(hashList)
	if err != nil {
		return err
	}
	defer objectsReader.Close()
	return writeDirectory(tarWriter, fileSystem, &fileSystem.DirectoryInode,
		".", objectsReader, make(map[uint64]struct{}))
}

func getOrderedObjectsList(fileSystem *filesystem.FileSystem) []hash.Hash {
	hashList := make([]hash.Hash, 0, len(fileSystem.InodeTable))
	inodeTable := make(map[uint64]struct{}, len(fileSystem.InodeTable))
	listObjects(&fileSystem.DirectoryInode, &hashList, inodeTable)
	return hashList
}

func listObjects(inode *filesystem.DirectoryInode, hashList *[]hash.Hash,
	inodeTable map[uint64]struct{}) {
	for _, entry := range inode.EntryList {
		if eInode, ok := entry.Inode().(*filesystem.RegularInode); ok {
			if eInode.Size > 0 {
				if _, ok := inodeTable[entry.InodeNumber]; !ok {
					*hashList = append(*hashList, eInode.Hash)
					inodeTable[entry.InodeNumber] = struct{}{}
				}
			}
		} else if eInode, ok := entry.Inode().(*filesystem.DirectoryInode); ok {
			listObjects(eInode, hashList, inodeTable)
		}
	}
}

func writeDirectory(tarWriter *tar.Writer, fileSystem *filesystem.FileSystem,
	inode *filesystem.DirectoryInode, dirname string,
	objectsReader objectserver.ObjectsReader,
	inodeTable map[uint64]struct{}) error {
	header := tar.Header{
		Name:     dirname + "/",
		Mode:     int64(inode.Mode),
		Uid:      int(inode.Uid),
		Gid:      int(inode.Gid),
		Typeflag: tar.TypeDir,
	}
	if err := tarWriter.WriteHeader(&header); err != nil {
		return err
	}
	for _, entry := range inode.EntryList {
		err := writeInode(tarWriter, fileSystem, entry.Inode(),
			dirname+"/"+entry.Name, entry.InodeNumber, objectsReader,
			inodeTable)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeRegularFile(tarWriter *tar.Writer, fileSystem *filesystem.FileSystem,
	inode *filesystem.RegularInode, name string, inodeNumber uint64,
	objectsReader objectserver.ObjectsReader,
	inodeTable map[uint64]struct{}) error {
	header := tar.Header{
		Name:     name,
		Mode:     int64(inode.Mode),
		Uid:      int(inode.Uid),
		Gid:      int(inode.Gid),
		Size:     int64(inode.Size),
		ModTime:  time.Unix(inode.MtimeSeconds, int64(inode.MtimeNanoSeconds)),
		Typeflag: tar.TypeReg,
	}
	err := writeHeader(tarWriter, fileSystem, &header, inodeNumber,
		inodeTable)
	if err != nil {
		return err
	}
	if header.Size > 0 {
		size, reader, err := objectsReader.NextObject()
		if err != nil {
			return err
		}
		defer reader.Close()
		if size != inode.Size {
			return fmt.Errorf("%s inode size: %u, object size: %d",
				name, inode.Size, size)
		}
		nCopied, err := io.Copy(tarWriter, reader)
		if err != nil {
			return err
		}
		if nCopied != int64(size) {
			return fmt.Errorf("nCopied: %d != size: %d", nCopied, size)
		}
	}
	return nil
}

func writeHeader(tarWriter *tar.Writer, fileSystem *filesystem.FileSystem,
	header *tar.Header, inum uint64, inodeTable map[uint64]struct{}) error {
	if _, ok := inodeTable[inum]; ok {
		header.Linkname = "." + fileSystem.InodeToFilenamesTable()[inum][0]
		header.Size = 0
		header.Typeflag = tar.TypeLink
	} else {
		inodeTable[inum] = struct{}{}
	}
	return tarWriter.WriteHeader(header)
}

func writeInode(tarWriter *tar.Writer, fileSystem *filesystem.FileSystem,
	inode filesystem.GenericInode, name string, inodeNumber uint64,
	objectsReader objectserver.ObjectsReader,
	inodeTable map[uint64]struct{}) error {
	var err error
	if eInode, ok := inode.(*filesystem.DirectoryInode); ok {
		err = writeDirectory(tarWriter, fileSystem, eInode, name, objectsReader,
			inodeTable)
	} else if eInode, ok := inode.(*filesystem.RegularInode); ok {
		err = writeRegularFile(tarWriter, fileSystem, eInode, name, inodeNumber,
			objectsReader, inodeTable)
	} else if eInode, ok := inode.(*filesystem.ComputedRegularInode); ok {
		err = writeRegularFile(tarWriter, fileSystem, &filesystem.RegularInode{
			Mode: eInode.Mode,
			Uid:  eInode.Uid,
			Gid:  eInode.Gid,
		}, name, inodeNumber, objectsReader, inodeTable)
	} else if eInode, ok := inode.(*filesystem.SpecialInode); ok {
		err = writeSpecial(tarWriter, fileSystem, eInode, name, inodeNumber,
			inodeTable)
	} else if eInode, ok := inode.(*filesystem.SymlinkInode); ok {
		err = writeSymlink(tarWriter, fileSystem, eInode, name, inodeNumber,
			inodeTable)
	}
	return err
}

func writeSpecial(tarWriter *tar.Writer, fileSystem *filesystem.FileSystem,
	inode *filesystem.SpecialInode, name string, inodeNumber uint64,
	inodeTable map[uint64]struct{}) error {
	header := tar.Header{
		Name:     name,
		Mode:     int64(inode.Mode),
		Uid:      int(inode.Uid),
		Gid:      int(inode.Gid),
		ModTime:  time.Unix(inode.MtimeSeconds, int64(inode.MtimeNanoSeconds)),
		Devmajor: int64(inode.Rdev >> 8),
		Devminor: int64(inode.Rdev & 0xff),
	}
	if inode.Mode&syscall.S_IFMT == syscall.S_IFCHR {
		header.Typeflag = tar.TypeChar
	} else if inode.Mode&syscall.S_IFMT == syscall.S_IFBLK {
		header.Typeflag = tar.TypeBlock
	} else if inode.Mode&syscall.S_IFMT == syscall.S_IFIFO {
		header.Typeflag = tar.TypeFifo
	} else {
		return fmt.Errorf("unsupported inode mode: %d", inode.Mode)
	}
	return writeHeader(tarWriter, fileSystem, &header, inodeNumber, inodeTable)
}

func writeSymlink(tarWriter *tar.Writer, fileSystem *filesystem.FileSystem,
	inode *filesystem.SymlinkInode, name string, inodeNumber uint64,
	inodeTable map[uint64]struct{}) error {
	header := tar.Header{
		Name:     name,
		Mode:     0777,
		Uid:      int(inode.Uid),
		Gid:      int(inode.Gid),
		Typeflag: tar.TypeSymlink,
		Linkname: inode.Symlink,
	}
	return writeHeader(tarWriter, fileSystem, &header, inodeNumber, inodeTable)
}
