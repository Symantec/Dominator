package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/fsutil"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
)

func getFileInImageSubcommand(args []string) {
	_, objectClient := getClients()
	var outFileName string
	if len(args) > 2 {
		outFileName = args[2]
	}
	err := getFileInImage(objectClient, args[0], args[1], outFileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting file in image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getFileInImage(objectClient *objectclient.ObjectClient, imageName,
	imageFile, outFileName string) error {
	fs, err := getTypedImage(imageName)
	if err != nil {
		return err
	}
	filenameToInodeTable := fs.FilenameToInodeTable()
	if inum, ok := filenameToInodeTable[imageFile]; !ok {
		return fmt.Errorf("file: \"%s\" not present in image", imageFile)
	} else if inode, ok := fs.InodeTable[inum]; !ok {
		return fmt.Errorf("inode: %d not present in image", inum)
	} else if inode, ok := inode.(*filesystem.RegularInode); !ok {
		return fmt.Errorf("file: \"%s\" is not a regular file", imageFile)
	} else {
		size, reader, err := objectClient.GetObject(inode.Hash)
		if err != nil {
			return err
		}
		defer reader.Close()
		if outFileName == "" {
			_, err := io.Copy(os.Stdout, reader)
			return err
		} else {
			return fsutil.CopyToFile(outFileName, filePerms, reader, size)
		}
	}
}
