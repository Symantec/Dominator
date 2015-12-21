package untar

import (
	"archive/tar"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"io"
	"path"
	"strings"
	"syscall"
)

type decoderData struct {
	nextInodeNumber uint64
	fileSystem      filesystem.FileSystem
	inodeTable      map[string]uint64
	directoryTable  map[string]*filesystem.DirectoryInode
}

func decode(tarReader *tar.Reader, dataHandler DataHandler,
	filter *filter.Filter) (*filesystem.FileSystem, error) {
	var decoderData decoderData
	decoderData.inodeTable = make(map[string]uint64)
	decoderData.directoryTable = make(map[string]*filesystem.DirectoryInode)
	fileSystem := &decoderData.fileSystem
	fileSystem.InodeTable = make(filesystem.InodeTable)
	// Create a default top-level directory which may be updated.
	decoderData.addInode("/", &fileSystem.DirectoryInode)
	fileSystem.DirectoryInode.Mode = syscall.S_IFDIR | syscall.S_IRWXU |
		syscall.S_IRGRP | syscall.S_IXGRP | syscall.S_IROTH | syscall.S_IXOTH
	decoderData.directoryTable["/"] = &fileSystem.DirectoryInode
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		header.Name = normaliseFilename(header.Name)
		if header.Name == "/.subd" ||
			strings.HasPrefix(header.Name, "/.subd/") {
			continue
		}
		if filter.Match(header.Name) {
			continue
		}
		err = decoderData.addHeader(tarReader, dataHandler, header)
		if err != nil {
			return nil, err
		}
	}
	delete(fileSystem.InodeTable, 0)
	fileSystem.DirectoryCount = uint64(len(decoderData.directoryTable))
	fileSystem.ComputeTotalDataBytes()
	sortDirectory(&fileSystem.DirectoryInode)
	return fileSystem, nil
}

func normaliseFilename(filename string) string {
	if filename[:2] == "./" {
		filename = filename[1:]
	} else if filename[0] != '/' {
		filename = "/" + filename
	}
	length := len(filename)
	if length > 1 && filename[length-1] == '/' {
		filename = filename[:length-1]
	}
	return filename
}

func (decoderData *decoderData) addHeader(tarReader *tar.Reader,
	dataHandler DataHandler, header *tar.Header) error {
	parentDir, ok := decoderData.directoryTable[path.Dir(header.Name)]
	if !ok {
		return errors.New(fmt.Sprintf(
			"No parent directory found for: %s", header.Name))
	}
	leafName := path.Base(header.Name)
	if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
		return decoderData.addRegularFile(tarReader, dataHandler, header,
			parentDir, leafName)
	} else if header.Typeflag == tar.TypeLink {
		return decoderData.addHardlink(header, parentDir, leafName)
	} else if header.Typeflag == tar.TypeSymlink {
		return decoderData.addSymlink(header, parentDir, leafName)
	} else if header.Typeflag == tar.TypeChar {
		return decoderData.addSpecialFile(header, parentDir, leafName)
	} else if header.Typeflag == tar.TypeBlock {
		return decoderData.addSpecialFile(header, parentDir, leafName)
	} else if header.Typeflag == tar.TypeDir {
		return decoderData.addDirectory(header, parentDir, leafName)
	} else if header.Typeflag == tar.TypeFifo {
		return decoderData.addSpecialFile(header, parentDir, leafName)
	} else {
		return errors.New(fmt.Sprintf("Unsupported file type: %v",
			header.Typeflag))
	}
	return nil
}

func (decoderData *decoderData) addRegularFile(tarReader *tar.Reader,
	dataHandler DataHandler, header *tar.Header,
	parent *filesystem.DirectoryInode, name string) error {
	var newInode filesystem.RegularInode
	newInode.Mode = filesystem.FileMode((header.Mode & ^syscall.S_IFMT) |
		syscall.S_IFREG)
	newInode.Uid = uint32(header.Uid)
	newInode.Gid = uint32(header.Gid)
	newInode.MtimeNanoSeconds = int32(header.ModTime.Nanosecond())
	newInode.MtimeSeconds = header.ModTime.Unix()
	newInode.Size = uint64(header.Size)
	if header.Size > 0 {
		var err error
		newInode.Hash, err = dataHandler.HandleData(tarReader,
			uint64(header.Size))
		if err != nil {
			return err
		}
	}
	decoderData.addEntry(parent, header.Name, name, &newInode)
	return nil
}

func (decoderData *decoderData) addDirectory(header *tar.Header,
	parent *filesystem.DirectoryInode, name string) error {
	var newInode filesystem.DirectoryInode
	newInode.Mode = filesystem.FileMode((header.Mode & ^syscall.S_IFMT) |
		syscall.S_IFDIR)
	newInode.Uid = uint32(header.Uid)
	newInode.Gid = uint32(header.Gid)
	if header.Name == "/" {
		*decoderData.directoryTable[header.Name] = newInode
		return nil
	}
	decoderData.addEntry(parent, header.Name, name, &newInode)
	decoderData.directoryTable[header.Name] = &newInode
	return nil
}

func (decoderData *decoderData) addHardlink(header *tar.Header,
	parent *filesystem.DirectoryInode, name string) error {
	header.Linkname = normaliseFilename(header.Linkname)
	if inum, ok := decoderData.inodeTable[header.Linkname]; ok {
		var newEntry filesystem.DirectoryEntry
		newEntry.Name = name
		newEntry.InodeNumber = inum
		parent.EntryList = append(parent.EntryList, &newEntry)
	} else {
		return errors.New(fmt.Sprintf("missing hardlink target: %s",
			header.Linkname))
	}
	return nil
}

func (decoderData *decoderData) addSymlink(header *tar.Header,
	parent *filesystem.DirectoryInode, name string) error {
	var newInode filesystem.SymlinkInode
	newInode.Uid = uint32(header.Uid)
	newInode.Gid = uint32(header.Gid)
	newInode.Symlink = header.Linkname
	decoderData.addEntry(parent, header.Name, name, &newInode)
	return nil
}

func (decoderData *decoderData) addSpecialFile(header *tar.Header,
	parent *filesystem.DirectoryInode, name string) error {
	var newInode filesystem.SpecialInode
	if header.Typeflag == tar.TypeChar {
		newInode.Mode = filesystem.FileMode((header.Mode & ^syscall.S_IFMT) |
			syscall.S_IFCHR)
	} else if header.Typeflag == tar.TypeBlock {
		newInode.Mode = filesystem.FileMode((header.Mode & ^syscall.S_IFMT) |
			syscall.S_IFBLK)
	} else if header.Typeflag == tar.TypeFifo {
		newInode.Mode = filesystem.FileMode((header.Mode & ^syscall.S_IFMT) |
			syscall.S_IFIFO)
	} else {
		return errors.New(fmt.Sprintf("unsupported type: %v", header.Typeflag))
	}
	newInode.Uid = uint32(header.Uid)
	newInode.Gid = uint32(header.Gid)
	newInode.MtimeNanoSeconds = int32(header.ModTime.Nanosecond())
	newInode.MtimeSeconds = header.ModTime.Unix()
	if header.Devminor > 255 {
		return errors.New(fmt.Sprintf("minor device number: %d too large",
			header.Devminor))
	}
	newInode.Rdev = uint64(header.Devmajor<<8 | header.Devminor)
	decoderData.addEntry(parent, header.Name, name, &newInode)
	return nil
}

func (decoderData *decoderData) addEntry(parent *filesystem.DirectoryInode,
	fullName, name string, inode filesystem.GenericInode) {
	var newEntry filesystem.DirectoryEntry
	newEntry.Name = name
	newEntry.InodeNumber = decoderData.nextInodeNumber
	newEntry.SetInode(inode)
	parent.EntryList = append(parent.EntryList, &newEntry)
	decoderData.addInode(fullName, inode)
}

func (decoderData *decoderData) addInode(fullName string,
	inode filesystem.GenericInode) {
	decoderData.inodeTable[fullName] = decoderData.nextInodeNumber
	decoderData.fileSystem.InodeTable[decoderData.nextInodeNumber] = inode
	decoderData.nextInodeNumber++
}
