package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type NumLinksTable map[uint64]int

type GenericInode interface {
	List(w io.Writer, name string, numLinksTable NumLinksTable,
		numLinks int) error
}

type InodeTable map[uint64]GenericInode
type InodeToFilenamesTable map[uint64][]string
type HashToInodesTable map[hash.Hash][]uint64

type FileSystem struct {
	InodeTable            InodeTable
	InodeToFilenamesTable InodeToFilenamesTable
	HashToInodesTable     HashToInodesTable
	NumRegularInodes      uint64
	TotalDataBytes        uint64
	DirectoryCount        uint64
	DirectoryInode
}

func (fs *FileSystem) RebuildInodePointers() {
	fs.rebuildInodePointers()
}

func (fs *FileSystem) BuildInodeToFilenamesTable() {
	fs.buildInodeToFilenamesTable()
}

func (fs *FileSystem) BuildHashToInodesTable() {
	fs.buildHashToInodesTable()
}

func (fs *FileSystem) ComputeTotalDataBytes() {
	fs.computeTotalDataBytes()
}

func (fs *FileSystem) List(w io.Writer) error {
	return fs.list(w)
}

type DirectoryInode struct {
	EntryList     []*DirectoryEntry
	EntriesByName map[string]*DirectoryEntry
	Mode          FileMode
	Uid           uint32
	Gid           uint32
}

func (directory *DirectoryInode) BuildEntryMap() {
	directory.buildEntryMap()
}

func (inode *DirectoryInode) List(w io.Writer, name string,
	numLinksTable NumLinksTable, numLinks int) error {
	return inode.list(w, name, numLinksTable, numLinks)
}

type DirectoryEntry struct {
	Name        string
	InodeNumber uint64
	inode       GenericInode // Keep private to avoid encoding/transmission.
}

func (dirent *DirectoryEntry) Inode() GenericInode {
	return dirent.inode
}

func (dirent *DirectoryEntry) SetInode(inode GenericInode) {
	dirent.inode = inode
}

func (dirent *DirectoryEntry) String() string {
	return dirent.Name
}

type RegularInode struct {
	Mode             FileMode
	Uid              uint32
	Gid              uint32
	MtimeNanoSeconds int32
	MtimeSeconds     int64
	Size             uint64
	Hash             hash.Hash
}

func (inode *RegularInode) List(w io.Writer, name string,
	numLinksTable NumLinksTable, numLinks int) error {
	return inode.list(w, name, numLinksTable, numLinks)
}

type SymlinkInode struct {
	Uid     uint32
	Gid     uint32
	Symlink string
}

func (inode *SymlinkInode) List(w io.Writer, name string,
	numLinksTable NumLinksTable, numLinks int) error {
	return inode.list(w, name, numLinksTable, numLinks)
}

type SpecialInode struct {
	Mode             FileMode
	Uid              uint32
	Gid              uint32
	MtimeNanoSeconds int32
	MtimeSeconds     int64
	Rdev             uint64
}

func (inode *SpecialInode) List(w io.Writer, name string,
	numLinksTable NumLinksTable, numLinks int) error {
	return inode.list(w, name, numLinksTable, numLinks)
}

type FileMode uint32

func (mode FileMode) String() string {
	return mode.string()
}

func CompareFileSystems(left, right *FileSystem, logWriter io.Writer) bool {
	return compareFileSystems(left, right, logWriter)
}

func CompareDirectoryInodes(left, right *DirectoryInode,
	logWriter io.Writer) bool {
	return compareDirectoryInodes(left, right, logWriter)
}

func CompareDirectoriesMetadata(left, right *DirectoryInode,
	logWriter io.Writer) bool {
	return compareDirectoriesMetadata(left, right, logWriter)
}

func CompareDirectoryEntries(left, right *DirectoryEntry,
	logWriter io.Writer) bool {
	return compareDirectoryEntries(left, right, logWriter)
}

func CompareInodes(left, right GenericInode, logWriter io.Writer) (
	sameType, sameMetadata, sameData bool) {
	return compareInodes(left, right, logWriter)
}

func CompareRegularInodes(left, right *RegularInode, logWriter io.Writer) bool {
	return compareRegularInodes(left, right, logWriter)
}

func CompareRegularInodesMetadata(left, right *RegularInode,
	logWriter io.Writer) bool {
	return compareRegularInodesMetadata(left, right, logWriter)
}

func CompareRegularInodesData(left, right *RegularInode,
	logWriter io.Writer) bool {
	return compareRegularInodesData(left, right, logWriter)
}

func CompareSymlinkInodes(left, right *SymlinkInode, logWriter io.Writer) bool {
	return compareSymlinkInodes(left, right, logWriter)
}

func CompareSymlinkInodesMetadata(left, right *SymlinkInode,
	logWriter io.Writer) bool {
	return compareSymlinkInodesMetadata(left, right, logWriter)
}

func CompareSymlinkInodesData(left, right *SymlinkInode,
	logWriter io.Writer) bool {
	return compareSymlinkInodesData(left, right, logWriter)
}

func CompareSpecialInodes(left, right *SpecialInode, logWriter io.Writer) bool {
	return compareSpecialInodes(left, right, logWriter)
}

func CompareSpecialInodesMetadata(left, right *SpecialInode,
	logWriter io.Writer) bool {
	return compareSpecialInodesMetadata(left, right, logWriter)
}

func CompareSpecialInodesData(left, right *SpecialInode,
	logWriter io.Writer) bool {
	return compareSpecialInodesData(left, right, logWriter)
}
