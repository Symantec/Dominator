package lib

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"testing"
)

func TestSameFile(t *testing.T) {
	request := makeUpdateRequest(testDataFile0(), testDataFile0())
	if len(request.PathsToDelete) != 0 {
		t.Errorf("number of paths to delete: %d != 0",
			len(request.PathsToDelete))
	}
}

func TestFileToDelete(t *testing.T) {
	request := makeUpdateRequest(testDataFile1(), testDataFile0())
	if len(request.PathsToDelete) != 1 {
		t.Errorf("number of paths to delete: %d != 1",
			len(request.PathsToDelete))
	}
}

func TestSameOnlyDirectory(t *testing.T) {
	request := makeUpdateRequest(testDataDirectory0(), testDataDirectory0())
	if len(request.PathsToDelete) != 0 {
		t.Errorf("number of paths to delete: %d != 0",
			len(request.PathsToDelete))
	}
}

func TestOnlyDirectoryToDelete(t *testing.T) {
	request := makeUpdateRequest(testDataDirectory2(), testDataDirectory0())
	if len(request.PathsToDelete) != 1 {
		t.Errorf("number of paths to delete: %d != 1",
			len(request.PathsToDelete))
	}
}

func TestExtraDirectoryToDelete(t *testing.T) {
	request := makeUpdateRequest(testDataDirectory0(), testDataDirectory01())
	if len(request.PathsToDelete) != 1 {
		t.Errorf("number of paths to delete: %d != 1",
			len(request.PathsToDelete))
	}
}

func makeUpdateRequest(imageFS *filesystem.FileSystem,
	subFS *filesystem.FileSystem) subproto.UpdateRequest {
	objectCache := make([]hash.Hash, 0, len(imageFS.InodeTable))
	for hashVal := range imageFS.HashToInodesTable() {
		objectCache = append(objectCache, hashVal)
	}
	imageFS.BuildEntryMap()
	if err := subFS.RebuildInodePointers(); err != nil {
		panic(err)
	}
	subFS.BuildEntryMap()
	if err := imageFS.RebuildInodePointers(); err != nil {
		panic(err)
	}
	subObj := Sub{FileSystem: subFS, ObjectCache: objectCache}
	var request subproto.UpdateRequest
	emptyFilter, _ := filter.New(nil)
	BuildUpdateRequest(subObj,
		&image.Image{FileSystem: imageFS, Filter: emptyFilter},
		&request,
		false, nil)
	return request
}

func testDataDirectory0() *filesystem.FileSystem {
	return &filesystem.FileSystem{
		InodeTable: filesystem.InodeTable{
			1: &filesystem.DirectoryInode{},
		},
		DirectoryInode: filesystem.DirectoryInode{
			EntryList: []*filesystem.DirectoryEntry{
				&filesystem.DirectoryEntry{
					Name:        "dir0",
					InodeNumber: 1,
				},
			},
		},
	}
}

func testDataDirectory01() *filesystem.FileSystem {
	return &filesystem.FileSystem{
		InodeTable: filesystem.InodeTable{
			1: &filesystem.DirectoryInode{},
			2: &filesystem.DirectoryInode{},
		},
		DirectoryInode: filesystem.DirectoryInode{
			EntryList: []*filesystem.DirectoryEntry{
				&filesystem.DirectoryEntry{
					Name:        "dir0",
					InodeNumber: 1,
				},
				&filesystem.DirectoryEntry{
					Name:        "dir1",
					InodeNumber: 2,
				},
			},
		},
	}
}

func testDataDirectory2() *filesystem.FileSystem {
	return &filesystem.FileSystem{
		InodeTable: filesystem.InodeTable{
			1: &filesystem.DirectoryInode{},
		},
		DirectoryInode: filesystem.DirectoryInode{
			EntryList: []*filesystem.DirectoryEntry{
				&filesystem.DirectoryEntry{
					Name:        "dir2",
					InodeNumber: 1,
				},
			},
		},
	}
}

func testDataFile0() *filesystem.FileSystem {
	return &filesystem.FileSystem{
		InodeTable: filesystem.InodeTable{
			1: &filesystem.RegularInode{Size: 100, Hash: hash0},
		},
		DirectoryInode: filesystem.DirectoryInode{
			EntryList: []*filesystem.DirectoryEntry{
				&filesystem.DirectoryEntry{
					Name:        "file0",
					InodeNumber: 1,
				},
			},
		},
	}
}

func testDataFile1() *filesystem.FileSystem {
	return &filesystem.FileSystem{
		InodeTable: filesystem.InodeTable{
			1: &filesystem.RegularInode{Size: 101, Hash: hash1},
		},
		DirectoryInode: filesystem.DirectoryInode{
			EntryList: []*filesystem.DirectoryEntry{
				&filesystem.DirectoryEntry{
					Name:        "file1",
					InodeNumber: 1,
				},
			},
		},
	}
}

var (
	hash0 hash.Hash = hash.Hash{0xde, 0xad}
	hash1 hash.Hash = hash.Hash{0xbe, 0xef}
)
