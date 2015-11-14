package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/objectserver"
	"io"
	"net/rpc"
	"os"
	"path"
	"syscall"
)

var dirPerms os.FileMode = syscall.S_IRWXU

func getImageSubcommand(imageClient *rpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	err := getImageAndWrite(imageClient, objectClient, args[0], args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting image\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getImageAndWrite(imageClient *rpc.Client,
	objectClient *objectclient.ObjectClient, name, dirname string) error {
	inodesDir := dirname + ".inodes"
	if err := os.Mkdir(inodesDir, dirPerms); err != nil {
		return err
	}
	defer os.RemoveAll(inodesDir)
	fs, err := getImage(imageClient, name)
	if err != nil {
		return err
	}
	var statfs syscall.Statfs_t
	if err := syscall.Statfs(inodesDir, &statfs); err != nil {
		return errors.New(fmt.Sprintf("Unable to Statfs: %s %s\n",
			inodesDir, err))
	}
	if fs.TotalDataBytes > uint64(statfs.Bsize)*statfs.Bfree {
		return errors.New("image will not fit on file-system")
	}
	hashes, inums, lengths := getHashes(fs)
	err = writeObjects(objectClient, hashes, inums, lengths, inodesDir)
	if err != nil {
		return err
	}
	if err := writeInodes(fs.InodeTable, inodesDir); err != nil {
		return err
	}
	if err = fs.DirectoryInode.Write(dirname); err != nil {
		return err
	}
	return buildTree(&fs.DirectoryInode, dirname, inodesDir)
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

func writeObjects(objectClient *objectclient.ObjectClient, hashes []hash.Hash,
	inums []uint64, lengths []uint64, inodesDir string) error {
	objectsReader, err := objectClient.GetObjects(hashes)
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting object reader: %s\n",
			err.Error()))
	}
	for index, hash := range hashes {
		err = writeObject(objectsReader, hash, inums[index], lengths[index],
			inodesDir)
		if err != nil {
			return err
		}
	}
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
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	var nCopied int64
	if nCopied, err = io.Copy(writer, reader); err != nil {
		return errors.New(fmt.Sprintf("error copying: %s", err.Error()))
	}
	if nCopied != int64(rlength) {
		return errors.New(fmt.Sprintf("expected length: %d, got: %d for: %x\n",
			rlength, nCopied, hash))
	}
	return nil
}

func writeInodes(inodeTable filesystem.InodeTable, inodesDir string) error {
	for inodeNumber, inode := range inodeTable {
		filename := path.Join(inodesDir, fmt.Sprintf("%d", inodeNumber))
		switch inode := inode.(type) {
		case *filesystem.RegularInode:
			if inode.Size < 1 {
				if _, err := os.Create(filename); err != nil {
					return err
				}
			}
			if err := inode.WriteMetadata(filename); err != nil {
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
	myPathName, inodesDir string) error {
	for _, dirent := range directory.EntryList {
		oldPath := path.Join(inodesDir, fmt.Sprintf("%d", dirent.InodeNumber))
		newPath := path.Join(myPathName, dirent.Name)
		if inode, ok := dirent.Inode().(*filesystem.DirectoryInode); ok {
			if err := os.Rename(oldPath, newPath); err != nil {
				return err
			}
			if err := buildTree(inode, newPath, inodesDir); err != nil {
				return err
			}
		} else {
			if err := os.Link(oldPath, newPath); err != nil {
				if !os.IsNotExist(err) {
					return err
				}
			}
		}
	}
	return nil
}
