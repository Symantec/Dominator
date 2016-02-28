package filesystem

import (
	"bytes"
	"fmt"
	"io"
	"syscall"
)

func compareFileSystems(left, right *FileSystem, logWriter io.Writer) bool {
	if len(left.InodeTable) != len(right.InodeTable) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d inodes\n",
				len(left.InodeTable), len(right.InodeTable))
		}
		return false
	}
	return compareDirectoryInodes(&left.DirectoryInode, &right.DirectoryInode,
		logWriter)
}

func compareDirectoryInodes(left, right *DirectoryInode,
	logWriter io.Writer) bool {
	if left == right {
		return true
	}
	if !compareDirectoriesMetadata(left, right, logWriter) {
		return false
	}
	if len(left.EntryList) != len(right.EntryList) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d entries\n",
				len(left.EntryList), len(right.EntryList))
		}
		return false
	}
	for index, leftEntry := range left.EntryList {
		if !compareDirectoryEntries(leftEntry, right.EntryList[index],
			logWriter) {
			return false
		}
	}
	return true
}

func compareDirectoriesMetadata(left, right *DirectoryInode,
	logWriter io.Writer) bool {
	if left.Mode != right.Mode {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mode: left vs. right: %o vs. %o\n",
				left.Mode, right.Mode)
		}
		return false
	}
	if left.Uid != right.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Uid: left vs. right: %d vs. %d\n",
				left.Uid, right.Uid)
		}
		return false
	}
	if left.Gid != right.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %d vs. %d\n",
				left.Gid, right.Gid)
		}
		return false
	}
	return true
}

func compareDirectoryEntries(left, right *DirectoryEntry,
	logWriter io.Writer) bool {
	if left == right {
		return true
	}
	if left.Name != right.Name {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "filename: left vs. right: %s vs. %s\n",
				left.Name, right.Name)
		}
		return false
	}
	switch left := left.inode.(type) {
	case *RegularInode:
		if right, ok := right.inode.(*RegularInode); ok {
			return compareRegularInodes(left, right, logWriter)
		}
	case *ComputedRegularInode:
		if right, ok := right.inode.(*ComputedRegularInode); ok {
			return compareComputedRegularInodes(left, right, logWriter)
		}
	case *SymlinkInode:
		if right, ok := right.inode.(*SymlinkInode); ok {
			return compareSymlinkInodes(left, right, logWriter)
		}
	case *SpecialInode:
		if right, ok := right.inode.(*SpecialInode); ok {
			return compareSpecialInodes(left, right, logWriter)
		}
	case *DirectoryInode:
		if right, ok := right.inode.(*DirectoryInode); ok {
			return compareDirectoryInodes(left, right, logWriter)
		}
	}
	if logWriter != nil {
		fmt.Fprintf(logWriter, "types: left vs. right: %s vs. %s\n",
			left.Name, right.Name)
	}
	return false
}

func compareInodes(left, right GenericInode, logWriter io.Writer) (
	sameType, sameMetadata, sameData bool) {
	if left == right {
		return true, true, true
	}
	switch left := left.(type) {
	case *RegularInode:
		if right, ok := right.(*RegularInode); ok {
			sameType = true
			sameMetadata = compareRegularInodesMetadata(left, right, logWriter)
			sameData = compareRegularInodesData(left, right, logWriter)
		}
	case *ComputedRegularInode:
		if right, ok := right.(*ComputedRegularInode); ok {
			sameType = true
			sameMetadata = compareComputedRegularInodesMetadata(left, right,
				logWriter)
			sameData = compareComputedRegularInodesData(left, right, logWriter)
		}
	case *SymlinkInode:
		if right, ok := right.(*SymlinkInode); ok {
			sameType = true
			sameMetadata = compareSymlinkInodesMetadata(left, right, logWriter)
			sameData = compareSymlinkInodesData(left, right, logWriter)
		}
	case *SpecialInode:
		if right, ok := right.(*SpecialInode); ok {
			sameType = true
			sameMetadata = compareSpecialInodesMetadata(left, right, logWriter)
			sameData = compareSpecialInodesData(left, right, logWriter)
		}
	case *DirectoryInode:
		if right, ok := right.(*DirectoryInode); ok {
			sameType = true
			sameMetadata = compareDirectoriesMetadata(left, right, logWriter)
		}
	default:
		panic(fmt.Sprintf("Unsupported entry type: %T", left))
	}
	if !sameType && logWriter != nil {
		fmt.Fprintln(logWriter, "types: left vs. right differ")
	}
	return
}

func compareRegularInodes(left, right *RegularInode, logWriter io.Writer) bool {
	if left == right {
		return true
	}
	if !compareRegularInodesMetadata(left, right, logWriter) {
		return false
	}
	return compareRegularInodesData(left, right, logWriter)
}

func compareRegularInodesMetadata(left, right *RegularInode,
	logWriter io.Writer) bool {
	if left.Mode != right.Mode {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mode: left vs. right: %o vs. %o\n",
				left.Mode, right.Mode)
		}
		return false
	}
	if left.Uid != right.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Uid: left vs. right: %d vs. %d\n",
				left.Uid, right.Uid)
		}
		return false
	}
	if left.Gid != right.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %d vs. %d\n",
				left.Gid, right.Gid)
		}
		return false
	}
	var leftMtime, rightMtime syscall.Timespec
	leftMtime.Sec = left.MtimeSeconds
	leftMtime.Nsec = int64(left.MtimeNanoSeconds)
	rightMtime.Sec = right.MtimeSeconds
	rightMtime.Nsec = int64(right.MtimeNanoSeconds)
	if leftMtime != rightMtime {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mtime: left vs. right: %v vs. %v\n",
				leftMtime, rightMtime)
		}
		return false
	}
	return true
}

func compareRegularInodesData(left, right *RegularInode,
	logWriter io.Writer) bool {
	if left.Size != right.Size {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Size: left vs. right: %d vs. %d\n",
				left.Size, right.Size)
		}
		return false
	}
	if left.Size > 0 {
		if bytes.Compare(left.Hash[:], right.Hash[:]) != 0 {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "hash: left vs. right: %x vs. %x\n",
					left.Hash, right.Hash)
			}
			return false
		}
	}
	return true
}

func compareComputedRegularInodes(left, right *ComputedRegularInode,
	logWriter io.Writer) bool {
	if left == right {
		return true
	}
	if !compareComputedRegularInodesMetadata(left, right, logWriter) {
		return false
	}
	return compareComputedRegularInodesData(left, right, logWriter)
}

func compareComputedRegularInodesMetadata(left, right *ComputedRegularInode,
	logWriter io.Writer) bool {
	if left.Mode != right.Mode {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mode: left vs. right: %o vs. %o\n",
				left.Mode, right.Mode)
		}
		return false
	}
	if left.Uid != right.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Uid: left vs. right: %d vs. %d\n",
				left.Uid, right.Uid)
		}
		return false
	}
	if left.Gid != right.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %d vs. %d\n",
				left.Gid, right.Gid)
		}
		return false
	}
	if left.Source != right.Source {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %s vs. %s\n",
				left.Source, right.Source)
		}
		return false
	}
	return true
}

func compareComputedRegularInodesData(left, right *ComputedRegularInode,
	logWriter io.Writer) bool {
	if left.Source != right.Source {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "data source: left vs. right: %s vs. %s\n",
				left.Source, right.Source)
		}
		return false
	}
	return true
}

func compareSymlinkInodes(left, right *SymlinkInode, logWriter io.Writer) bool {
	if left == right {
		return true
	}
	if !compareSymlinkInodesMetadata(left, right, logWriter) {
		return false
	}
	return compareSymlinkInodesData(left, right, logWriter)
}

func compareSymlinkInodesMetadata(left, right *SymlinkInode,
	logWriter io.Writer) bool {
	if left.Uid != right.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Uid: left vs. right: %d vs. %d\n",
				left.Uid, right.Uid)
		}
		return false
	}
	if left.Gid != right.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %d vs. %d\n",
				left.Gid, right.Gid)
		}
		return false
	}
	return true
}

func compareSymlinkInodesData(left, right *SymlinkInode,
	logWriter io.Writer) bool {
	if left.Symlink != right.Symlink {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "symlink: left vs. right: %s vs. %s\n",
				left.Symlink, right.Symlink)
		}
		return false
	}
	return true
}

func compareSpecialInodes(left, right *SpecialInode, logWriter io.Writer) bool {
	if left == right {
		return true
	}
	if !compareSpecialInodesMetadata(left, right, logWriter) {
		return false
	}
	return compareSpecialInodesData(left, right, logWriter)
}

func compareSpecialInodesMetadata(left, right *SpecialInode,
	logWriter io.Writer) bool {
	if left == right {
		return true
	}
	if left.Mode != right.Mode {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mode: left vs. right: %o vs. %o\n",
				left.Mode, right.Mode)
		}
		return false
	}
	if left.Uid != right.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Uid: left vs. right: %d vs. %d\n",
				left.Uid, right.Uid)
		}
		return false
	}
	if left.Gid != right.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %d vs. %d\n",
				left.Gid, right.Gid)
		}
		return false
	}
	var leftMtime, rightMtime syscall.Timespec
	leftMtime.Sec = left.MtimeSeconds
	leftMtime.Nsec = int64(left.MtimeNanoSeconds)
	rightMtime.Sec = right.MtimeSeconds
	rightMtime.Nsec = int64(right.MtimeNanoSeconds)
	if leftMtime != rightMtime {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mtime: left vs. right: %v vs. %v\n",
				leftMtime, rightMtime)
		}
		return false
	}
	return true
}

func compareSpecialInodesData(left, right *SpecialInode,
	logWriter io.Writer) bool {
	if left.Mode&syscall.S_IFMT == syscall.S_IFBLK ||
		left.Mode&syscall.S_IFMT == syscall.S_IFCHR {
		if left.Rdev != right.Rdev {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "Rdev: left vs. right: %#x vs. %#x\n",
					left.Rdev, right.Rdev)
			}
			return false
		}
	}
	return true
}
