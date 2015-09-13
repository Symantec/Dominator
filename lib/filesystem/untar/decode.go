package untar

import (
	"archive/tar"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"io"
	"io/ioutil"
	"path"
	"syscall"
)

type decoderData struct {
	nextInodeNumber   uint64
	fileSystem        filesystem.FileSystem
	regularInodeTable map[string]uint64
	symlinkInodeTable map[string]uint64
	inodeTable        map[string]uint64
	directoryTable    map[string]*filesystem.Directory
}

func decode(tarReader *tar.Reader, dataHandler DataHandler,
	filter *filter.Filter) (*filesystem.FileSystem, error) {
	var decoderData decoderData
	decoderData.regularInodeTable = make(map[string]uint64)
	decoderData.symlinkInodeTable = make(map[string]uint64)
	decoderData.inodeTable = make(map[string]uint64)
	decoderData.directoryTable = make(map[string]*filesystem.Directory)
	fileSystem := &decoderData.fileSystem
	fileSystem.RegularInodeTable = make(filesystem.RegularInodeTable)
	fileSystem.SymlinkInodeTable = make(filesystem.SymlinkInodeTable)
	fileSystem.InodeTable = make(filesystem.InodeTable)
	// Create a container directory for the top-level directory.
	var containerDir filesystem.Directory
	decoderData.directoryTable["/"] = &containerDir
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		header.Name = normaliseFilename(header.Name)
		if filter.Match(header.Name) {
			continue
		}
		err = decoderData.addHeader(tarReader, dataHandler, header)
		if err != nil {
			return nil, err
		}
	}
	fileSystem.Directory = *containerDir.DirectoryList[0]
	sortDirectory(&fileSystem.Directory)
	fileSystem.DirectoryCount = uint64(len(decoderData.directoryTable))
	fileSystem.RebuildInodePointers()
	fileSystem.ComputeTotalDataBytes()
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
		return decoderData.addFile(header, parentDir, leafName)
	} else if header.Typeflag == tar.TypeBlock {
		return decoderData.addFile(header, parentDir, leafName)
	} else if header.Typeflag == tar.TypeDir {
		return decoderData.addDirectory(header, parentDir, leafName)
	} else if header.Typeflag == tar.TypeFifo {
		return decoderData.addFile(header, parentDir, leafName)
	} else {
		return errors.New(fmt.Sprintf("Unsupported file type: %v",
			header.Typeflag))
	}
	return nil
}

func (decoderData *decoderData) addRegularFile(tarReader *tar.Reader,
	dataHandler DataHandler, header *tar.Header, parent *filesystem.Directory,
	name string) error {
	var newInode filesystem.RegularInode
	newInode.Mode = filesystem.FileMode((header.Mode & ^syscall.S_IFMT) |
		syscall.S_IFREG)
	newInode.Uid = uint32(header.Uid)
	newInode.Gid = uint32(header.Gid)
	newInode.MtimeNanoSeconds = int32(header.ModTime.Nanosecond())
	newInode.MtimeSeconds = header.ModTime.Unix()
	newInode.Size = uint64(header.Size)
	if header.Size > 0 {
		data, err := ioutil.ReadAll(tarReader)
		if err != nil {
			return errors.New("error reading file data" + err.Error())
		}
		if int64(len(data)) != header.Size {
			return errors.New(fmt.Sprintf(
				"failed to read file data, wanted: %d, got: %d bytes",
				header.Size, len(data)))
		}
		newInode.Hash, err = dataHandler.HandleData(data)
		if err != nil {
			return err
		}
	}
	decoderData.regularInodeTable[header.Name] = decoderData.nextInodeNumber
	decoderData.fileSystem.RegularInodeTable[decoderData.nextInodeNumber] =
		&newInode
	var newEntry filesystem.RegularFile
	newEntry.Name = name
	newEntry.InodeNumber = decoderData.nextInodeNumber
	parent.RegularFileList = append(parent.RegularFileList, &newEntry)
	decoderData.nextInodeNumber++
	return nil
}

func (decoderData *decoderData) addDirectory(header *tar.Header,
	parent *filesystem.Directory, name string) error {
	var newEntry filesystem.Directory
	newEntry.Name = name
	newEntry.Mode = filesystem.FileMode((header.Mode & ^syscall.S_IFMT) |
		syscall.S_IFDIR)
	newEntry.Uid = uint32(header.Uid)
	newEntry.Gid = uint32(header.Gid)
	parent.DirectoryList = append(parent.DirectoryList, &newEntry)
	decoderData.directoryTable[header.Name] = &newEntry
	return nil
}

func (decoderData *decoderData) addHardlink(header *tar.Header,
	parent *filesystem.Directory, name string) error {
	header.Linkname = normaliseFilename(header.Linkname)
	if inum, ok := decoderData.regularInodeTable[header.Linkname]; ok {
		var newEntry filesystem.RegularFile
		newEntry.Name = name
		newEntry.InodeNumber = inum
		parent.RegularFileList = append(parent.RegularFileList, &newEntry)
	} else if inum, ok := decoderData.symlinkInodeTable[header.Linkname]; ok {
		var newEntry filesystem.Symlink
		newEntry.Name = name
		newEntry.InodeNumber = inum
		parent.SymlinkList = append(parent.SymlinkList, &newEntry)
	} else if inum, ok := decoderData.inodeTable[header.Linkname]; ok {
		var newEntry filesystem.File
		newEntry.Name = name
		newEntry.InodeNumber = inum
		parent.FileList = append(parent.FileList, &newEntry)
	} else {
		return errors.New(fmt.Sprintf("missing hardlink target: %s",
			header.Linkname))
	}
	return nil
}

func (decoderData *decoderData) addSymlink(header *tar.Header,
	parent *filesystem.Directory, name string) error {
	var newInode filesystem.SymlinkInode
	newInode.Uid = uint32(header.Uid)
	newInode.Gid = uint32(header.Gid)
	newInode.Symlink = header.Linkname
	decoderData.symlinkInodeTable[header.Name] = decoderData.nextInodeNumber
	decoderData.fileSystem.SymlinkInodeTable[decoderData.nextInodeNumber] =
		&newInode
	var newEntry filesystem.Symlink
	newEntry.Name = name
	newEntry.InodeNumber = decoderData.nextInodeNumber
	parent.SymlinkList = append(parent.SymlinkList, &newEntry)
	decoderData.nextInodeNumber++
	return nil
}

func (decoderData *decoderData) addFile(header *tar.Header,
	parent *filesystem.Directory, name string) error {
	var newInode filesystem.Inode
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
	decoderData.inodeTable[header.Name] = decoderData.nextInodeNumber
	decoderData.fileSystem.InodeTable[decoderData.nextInodeNumber] =
		&newInode
	var newEntry filesystem.File
	newEntry.Name = name
	newEntry.InodeNumber = decoderData.nextInodeNumber
	parent.FileList = append(parent.FileList, &newEntry)
	decoderData.nextInodeNumber++
	return nil
}
