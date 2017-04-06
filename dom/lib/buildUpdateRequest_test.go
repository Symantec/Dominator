package lib

import (
	"encoding/json"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/log/testlogger"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"reflect"
	"testing"
)

type fileInfo struct {
	size    uint64
	hashVal hash.Hash
	uid     uint32
}

func TestSameFile(t *testing.T) {
	request := makeUpdateRequest(t, testDataFile0(0), testDataFile0(0))
	if len(request.PathsToDelete) != 0 {
		t.Errorf("number of paths to delete: %d != 0",
			len(request.PathsToDelete))
	}
}

func TestFileToDelete(t *testing.T) {
	request := makeUpdateRequest(t, testDataFile1(0), testDataFile0(0))
	if len(request.PathsToDelete) != 1 {
		t.Errorf("number of paths to delete: %d != 1",
			len(request.PathsToDelete))
	}
}

func TestFileToChange(t *testing.T) {
	request := makeUpdateRequest(t, testDataFile0(0), testDataFile0(1))
	if reflect.DeepEqual(request, subproto.UpdateRequest{}) {
		t.Error("Inode not being changed")
	}
	if len(request.InodesToChange) != 1 {
		t.Error("Inode not being changed")
	}
}

func TestSameOnlyDirectory(t *testing.T) {
	request := makeUpdateRequest(t, testDataDirectory0(), testDataDirectory0())
	if len(request.PathsToDelete) != 0 {
		t.Errorf("number of paths to delete: %d != 0",
			len(request.PathsToDelete))
	}
}

func TestOnlyDirectoryToDelete(t *testing.T) {
	request := makeUpdateRequest(t, testDataDirectory2(), testDataDirectory0())
	if len(request.PathsToDelete) != 1 {
		t.Errorf("number of paths to delete: %d != 1",
			len(request.PathsToDelete))
	}
}

func TestExtraDirectoryToDelete(t *testing.T) {
	request := makeUpdateRequest(t, testDataDirectory0(), testDataDirectory01())
	if len(request.PathsToDelete) != 1 {
		t.Errorf("number of paths to delete: %d != 1",
			len(request.PathsToDelete))
	}
}

func TestLinkFiles(t *testing.T) {
	request := makeUpdateRequest(t, testDataLinkedFiles(2),
		testDataDuplicateFiles())
	if len(request.HardlinksToMake) != 1 {
		t.Error("File not being linked")
	}
	req := subproto.UpdateRequest{
		HardlinksToMake: request.HardlinksToMake,
	}
	if !reflect.DeepEqual(request, req) {
		t.Error("Unexpected changes being made")
	}
}

func TestSplitHardlinks(t *testing.T) {
	request := makeUpdateRequest(t, testDataDuplicateFiles(),
		testDataLinkedFiles(2))
	if len(request.FilesToCopyToCache) != 1 {
		t.Error("File not being copied to cache")
	}
	if len(request.InodesToMake) != 1 {
		t.Error("Inode not being created")
	}
	req := subproto.UpdateRequest{
		FilesToCopyToCache: request.FilesToCopyToCache,
		InodesToMake:       request.InodesToMake,
	}
	if !reflect.DeepEqual(request, req) {
		t.Error("Unexpected changes being made")
	}
}

func TestSplitHashes1(t *testing.T) {
	request := makeUpdateRequest(t,
		testDataMulti(
			[]fileInfo{{101, hash1, 1}, {101, hash1, 0}},
			[]uint64{1, 2}),
		testDataLinkedFiles(2))
	if len(request.FilesToCopyToCache) != 1 {
		t.Error("File not being copied to cache")
	}
	if request.FilesToCopyToCache[0].DoHardlink {
		t.Errorf("%s is being hardlinked to cache",
			request.FilesToCopyToCache[0].Name)
	}
	if len(request.InodesToMake) != 1 {
		t.Error("Inode not being created")
	}
	if request.InodesToMake[0].Name != "/file1" {
		t.Error("/file1 not being created")
	}
	req := subproto.UpdateRequest{
		FilesToCopyToCache: request.FilesToCopyToCache,
		InodesToMake:       request.InodesToMake,
	}
	if !reflect.DeepEqual(request, req) {
		t.Error("Unexpected changes being made")
	}
}

func TestSplitHashes2(t *testing.T) {
	request := makeUpdateRequest(t,
		testDataMulti(
			[]fileInfo{{101, hash1, 0}, {101, hash1, 1}},
			[]uint64{1, 2}),
		testDataLinkedFiles(2))
	if len(request.FilesToCopyToCache) != 1 {
		t.Error("File not being copied to cache")
	}
	if len(request.InodesToMake) != 1 {
		t.Error("Inode not being created")
	}
	req := subproto.UpdateRequest{
		FilesToCopyToCache: request.FilesToCopyToCache,
		InodesToMake:       request.InodesToMake,
	}
	if !reflect.DeepEqual(request, req) {
		t.Error("Unexpected changes being made")
	}
}

func TestSplitHashes3(t *testing.T) {
	request := makeUpdateRequest(t,
		testDataMulti(
			[]fileInfo{{101, hash1, 1}, {101, hash1, 0}},
			[]uint64{1, 2, 2}),
		testDataLinkedFiles(3))
	if len(request.FilesToCopyToCache) != 1 {
		t.Error("File not being copied to cache")
	}
	if request.FilesToCopyToCache[0].DoHardlink {
		t.Errorf("%s is being hardlinked to cache",
			request.FilesToCopyToCache[0].Name)
	}
	if len(request.InodesToMake) != 1 {
		t.Error("Inode not being created")
	}
	if request.InodesToMake[0].Name != "/file1" {
		t.Error("/file1 not being created")
	}
	req := subproto.UpdateRequest{
		FilesToCopyToCache: request.FilesToCopyToCache,
		InodesToMake:       request.InodesToMake,
	}
	if !reflect.DeepEqual(request, req) {
		t.Error("Unexpected changes being made")
	}
}

func TestSplitHashes4(t *testing.T) {
	request := makeUpdateRequest(t,
		testDataMulti(
			[]fileInfo{{101, hash1, 1}, {101, hash1, 0}, {101, hash1, 0}},
			[]uint64{1, 2, 2, 1, 3}),
		testDataLinkedFiles(4))
	if len(request.FilesToCopyToCache) != 1 {
		t.Error("File not being copied to cache")
	}
	if request.FilesToCopyToCache[0].DoHardlink {
		t.Errorf("%s is being hardlinked to cache",
			request.FilesToCopyToCache[0].Name)
	}
	if len(request.InodesToMake) != 2 {
		t.Error("Inodes not being created")
	}
	if request.InodesToMake[0].Name != "/file1" {
		t.Error("/file1 not being created")
	}
	if request.InodesToMake[1].Name != "/file5" {
		t.Error("/file5 not being created")
	}
	if len(request.HardlinksToMake) != 1 {
		t.Error("File not being linked")
	}
	if request.HardlinksToMake[0].NewLink != "/file4" ||
		request.HardlinksToMake[0].Target != "/file1" {
		t.Errorf("/file4 not being linked to /file1")
	}
	if len(request.MultiplyUsedObjects) != 1 {
		t.Error("Object not being duplicated")
	}
	req := subproto.UpdateRequest{
		FilesToCopyToCache:  request.FilesToCopyToCache,
		InodesToMake:        request.InodesToMake,
		HardlinksToMake:     request.HardlinksToMake,
		MultiplyUsedObjects: request.MultiplyUsedObjects,
	}
	if !reflect.DeepEqual(request, req) {
		t.Error("Unexpected changes being made")
	}
}

func TestSplitHashes5(t *testing.T) {
	request := makeUpdateRequest(t,
		testDataMulti(
			[]fileInfo{{101, hash1, 1}, {101, hash1, 0}, {101, hash1, 1}},
			[]uint64{1, 2, 2, 1, 3}),
		testDataMulti(
			[]fileInfo{{101, hash1, 1}, {101, hash1, 0}, {101, hash1, 0}},
			[]uint64{1, 2, 1, 2, 3}))
	if len(request.HardlinksToMake) != 2 {
		t.Error("#HardlinksToMake: %d != 2", len(request.HardlinksToMake))
	}
	if request.HardlinksToMake[0].NewLink != "/file3" ||
		request.HardlinksToMake[0].Target != "/file2" {
		t.Errorf("/file3 not being linked to /file2")
	}
	if request.HardlinksToMake[1].NewLink != "/file4" ||
		request.HardlinksToMake[1].Target != "/file1" {
		t.Errorf("/file4 not being linked to /file1")
	}
	if len(request.InodesToChange) < 1 {
		t.Error("Inode not being changed")
	}
	if len(request.InodesToChange) > 1 {
		t.Error("Too many inodes being changed")
	}
	if request.InodesToChange[0].Name != "/file5" {
		t.Error("/file5 not being changed")
	}
	req := subproto.UpdateRequest{
		HardlinksToMake: request.HardlinksToMake,
		InodesToChange:  request.InodesToChange,
	}
	if !reflect.DeepEqual(request, req) {
		t.Error("Unexpected changes being made")
	}
}

func makeUpdateRequest(t *testing.T, imageFS *filesystem.FileSystem,
	subFS *filesystem.FileSystem) subproto.UpdateRequest {
	fetchedObjects := make(map[hash.Hash]struct{}, len(imageFS.InodeTable))
	for hashVal := range imageFS.HashToInodesTable() {
		fetchedObjects[hashVal] = struct{}{}
	}
	for hashVal := range subFS.HashToInodesTable() {
		delete(fetchedObjects, hashVal)
	}
	objectCache := make([]hash.Hash, 0, len(fetchedObjects))
	for hashVal := range fetchedObjects {
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
		&request, false, false, testlogger.New(t))
	reqTxt, err := json.MarshalIndent(request, "", "    ")
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("%s", reqTxt)
	}
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

func testDataFile0(uid uint32) *filesystem.FileSystem {
	return &filesystem.FileSystem{
		InodeTable: filesystem.InodeTable{
			1: &filesystem.RegularInode{Size: 100, Hash: hash0, Uid: uid},
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

func testDataFile1(uid uint32) *filesystem.FileSystem {
	return &filesystem.FileSystem{
		InodeTable: filesystem.InodeTable{
			1: &filesystem.RegularInode{Size: 101, Hash: hash1, Uid: uid},
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

func testDataDuplicateFiles() *filesystem.FileSystem {
	return &filesystem.FileSystem{
		InodeTable: filesystem.InodeTable{
			1: &filesystem.RegularInode{Size: 101, Hash: hash1},
			2: &filesystem.RegularInode{Size: 101, Hash: hash1},
		},
		DirectoryInode: filesystem.DirectoryInode{
			EntryList: []*filesystem.DirectoryEntry{
				&filesystem.DirectoryEntry{
					Name:        "file1",
					InodeNumber: 1,
				},
				&filesystem.DirectoryEntry{
					Name:        "file2",
					InodeNumber: 2,
				},
			},
		},
	}
}

func testDataLinkedFiles(nFiles int) *filesystem.FileSystem {
	entries := make([]*filesystem.DirectoryEntry, 0, nFiles)
	for i := 0; i < nFiles; i++ {
		entries = append(entries, &filesystem.DirectoryEntry{
			Name:        fmt.Sprintf("file%d", i+1),
			InodeNumber: 1,
		})
	}
	return &filesystem.FileSystem{
		InodeTable: filesystem.InodeTable{
			1: &filesystem.RegularInode{Size: 101, Hash: hash1},
		},
		DirectoryInode: filesystem.DirectoryInode{EntryList: entries},
	}
}

func testDataMulti(fInfos []fileInfo, files []uint64) *filesystem.FileSystem {
	inodes := make(filesystem.InodeTable, len(fInfos))
	for i := uint64(0); i < uint64(len(fInfos)); i++ {
		fi := fInfos[i]
		inodes[i+1] = &filesystem.RegularInode{
			Size: fi.size,
			Hash: fi.hashVal,
			Uid:  fi.uid,
		}
	}
	entries := make([]*filesystem.DirectoryEntry, 0, len(files))
	for i := 0; i < len(files); i++ {
		entries = append(entries, &filesystem.DirectoryEntry{
			Name:        fmt.Sprintf("file%d", i+1),
			InodeNumber: files[i],
		})
	}
	return &filesystem.FileSystem{
		InodeTable:     inodes,
		DirectoryInode: filesystem.DirectoryInode{EntryList: entries},
	}
}

var (
	hash0 hash.Hash = hash.Hash{0xde, 0xad}
	hash1 hash.Hash = hash.Hash{0xbe, 0xef}
)
