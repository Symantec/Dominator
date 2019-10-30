package util

import (
	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
)

func copyMtimes(sourceFs, destFs *filesystem.FileSystem) {
	sourceFilenameToInodeTable := sourceFs.FilenameToInodeTable()
	destInodeToFilenamesTable := destFs.InodeToFilenamesTable()
	for inum, destInode := range destFs.InodeTable {
		filenames := destInodeToFilenamesTable[inum]
		var sourceInode filesystem.GenericInode
		for _, filename := range filenames { // Search for source inode.
			if sourceInum, ok := sourceFilenameToInodeTable[filename]; ok {
				sourceInode = sourceFs.InodeTable[sourceInum]
				break
			}
		}
		if sourceInode == nil {
			continue
		}
		if destInode, ok := destInode.(*filesystem.RegularInode); ok {
			if sourceInode, ok := sourceInode.(*filesystem.RegularInode); ok {
				newIno := *destInode
				newIno.MtimeNanoSeconds = sourceInode.MtimeNanoSeconds
				newIno.MtimeSeconds = sourceInode.MtimeSeconds
				if filesystem.CompareRegularInodes(&newIno, sourceInode, nil) {
					destFs.InodeTable[inum] = &newIno
				}
			}
		}
		if destInode, ok := destInode.(*filesystem.SpecialInode); ok {
			if sourceInode, ok := sourceInode.(*filesystem.SpecialInode); ok {
				newIno := *destInode
				newIno.MtimeNanoSeconds = sourceInode.MtimeNanoSeconds
				newIno.MtimeSeconds = sourceInode.MtimeSeconds
				if filesystem.CompareSpecialInodes(&newIno, sourceInode, nil) {
					destFs.InodeTable[inum] = &newIno
				}
			}
		}
	}
}
