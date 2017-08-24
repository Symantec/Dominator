package util

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectserver"
	"os"
	"path"
	"syscall"
	"time"
)

const (
	dirPerms  = syscall.S_IRWXU
	filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP
)

func createFile(filename string) error {
	if file, err := os.Create(filename); err != nil {
		return err
	} else {
		// Don't wait for finaliser to close, otherwise we can have too many
		// open files.
		file.Close()
		return nil
	}
}

func unpack(fs *filesystem.FileSystem, objectsGetter objectserver.ObjectsGetter,
	dirname string, logger log.Logger) error {
	inodesDir := dirname + ".inodes"
	if err := os.Mkdir(inodesDir, dirPerms); err != nil {
		return err
	}
	defer os.RemoveAll(inodesDir)
	var statfs syscall.Statfs_t
	if err := syscall.Statfs(inodesDir, &statfs); err != nil {
		return errors.New(fmt.Sprintf("Unable to Statfs: %s %s\n",
			inodesDir, err))
	}
	if fs.TotalDataBytes > uint64(statfs.Bsize)*statfs.Bfree {
		return errors.New("image will not fit on file-system")
	}
	hashes, inums, lengths := getHashes(fs)
	err := writeObjects(objectsGetter, hashes, inums, lengths, inodesDir,
		logger)
	if err != nil {
		return err
	}
	startWriteTime := time.Now()
	if err := writeInodes(fs.InodeTable, inodesDir); err != nil {
		return err
	}
	if err = fs.DirectoryInode.Write(dirname); err != nil {
		return err
	}
	startBuildTime := time.Now()
	writeDuration := startBuildTime.Sub(startWriteTime)
	err = buildTree(&fs.DirectoryInode, dirname, "", inodesDir)
	if err != nil {
		return err
	}
	buildDuration := time.Since(startBuildTime)
	logger.Printf("Unpacked file-system: made inodes in %s, built tree in %s\n",
		format.Duration(writeDuration), format.Duration(buildDuration))
	return nil
}

func getHashes(fs *filesystem.FileSystem) ([]hash.Hash, []uint64, []uint64) {
	hashes := make([]hash.Hash, 0, fs.NumRegularInodes)
	inums := make([]uint64, 0, fs.NumRegularInodes)
	lengths := make([]uint64, 0, fs.NumRegularInodes)
	for inum, inode := range fs.InodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			if inode.Size > 0 {
				hashes = append(hashes, inode.Hash)
				inums = append(inums, inum)
				lengths = append(lengths, inode.Size)
			}
		}
	}
	return hashes, inums, lengths
}

func writeObjects(objectsGetter objectserver.ObjectsGetter, hashes []hash.Hash,
	inums []uint64, lengths []uint64, inodesDir string,
	logger log.Logger) error {
	startTime := time.Now()
	objectsReader, err := objectsGetter.GetObjects(hashes)
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting object reader: %s\n",
			err.Error()))
	}
	defer objectsReader.Close()
	var totalLength uint64
	for index, hash := range hashes {
		err = writeObject(objectsReader, hash, inums[index], lengths[index],
			inodesDir)
		if err != nil {
			return err
		}
		totalLength += lengths[index]
	}
	duration := time.Since(startTime)
	speed := uint64(float64(totalLength) / duration.Seconds())
	logger.Printf("Copied %d objects (%s) in %s (%s/s)\n",
		len(hashes), format.FormatBytes(totalLength), format.Duration(duration),
		format.FormatBytes(speed))
	return nil
}

func writeObject(objectsReader objectserver.ObjectsReader, hash hash.Hash,
	inodeNumber uint64, length uint64, inodesDir string) error {
	rlength, reader, err := objectsReader.NextObject()
	if err != nil {
		return err
	}
	defer reader.Close()
	if rlength != length {
		return errors.New("mismatched lengths")
	}
	filename := path.Join(inodesDir, fmt.Sprintf("%d", inodeNumber))
	return fsutil.CopyToFile(filename, filePerms, reader, rlength)
}

func writeInodes(inodeTable filesystem.InodeTable, inodesDir string) error {
	for inodeNumber, inode := range inodeTable {
		filename := path.Join(inodesDir, fmt.Sprintf("%d", inodeNumber))
		switch inode := inode.(type) {
		case *filesystem.RegularInode:
			if inode.Size < 1 {
				if err := createFile(filename); err != nil {
					return err
				}
			}
			if err := inode.WriteMetadata(filename); err != nil {
				return err
			}
		case *filesystem.ComputedRegularInode:
			if err := createFile(filename); err != nil {
				return err
			}
			tmpInode := &filesystem.RegularInode{
				Mode: inode.Mode,
				Uid:  inode.Uid,
				Gid:  inode.Gid,
			}
			if err := tmpInode.WriteMetadata(filename); err != nil {
				return err
			}
		case *filesystem.SymlinkInode:
			if err := inode.Write(filename); err != nil {
				return err
			}
		case *filesystem.SpecialInode:
			if err := inode.Write(filename); err != nil {
				return err
			}
		case *filesystem.DirectoryInode:
			if err := inode.Write(filename); err != nil {
				return err
			}
		default:
			return errors.New("unsupported inode type")
		}
	}
	return nil
}

func buildTree(directory *filesystem.DirectoryInode,
	rootDir, mySubPathName, inodesDir string) error {
	for _, dirent := range directory.EntryList {
		oldPath := path.Join(inodesDir, fmt.Sprintf("%d", dirent.InodeNumber))
		newSubPath := path.Join(mySubPathName, dirent.Name)
		newFullPath := path.Join(rootDir, newSubPath)
		if inode := dirent.Inode(); inode == nil {
			panic("no inode pointer for: " + newSubPath)
		} else if inode, ok := inode.(*filesystem.DirectoryInode); ok {
			if err := os.Rename(oldPath, newFullPath); err != nil {
				return err
			}
			err := buildTree(inode, rootDir, newSubPath, inodesDir)
			if err != nil {
				return err
			}
		} else {
			if err := os.Link(oldPath, newFullPath); err != nil {
				if !os.IsNotExist(err) {
					return err
				}
			}
		}
	}
	return nil
}
