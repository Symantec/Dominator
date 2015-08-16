package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type RegularInodeTable map[uint64]*RegularInode
type SymlinkInodeTable map[uint64]*SymlinkInode
type InodeTable map[uint64]*Inode

type FileSystem struct {
	RegularInodeTable RegularInodeTable
	SymlinkInodeTable SymlinkInodeTable
	InodeTable        InodeTable
	TotalDataBytes    uint64
	DirectoryCount    uint64
	Directory
}

func (fs *FileSystem) RebuildPointers() {
	fs.rebuildPointers()
}

func (fs *FileSystem) ComputeTotalDataBytes() {
	fs.computeTotalDataBytes()
}

func (fs *FileSystem) DebugWrite(w io.Writer, prefix string) error {
	return fs.debugWrite(w, prefix)
}

type Directory struct {
	Name            string
	RegularFileList []*RegularFile
	SymlinkList     []*Symlink
	FileList        []*File
	DirectoryList   []*Directory
	Mode            uint32
	Uid             uint32
	Gid             uint32
}

func (directory *Directory) String() string {
	return directory.Name
}

func (directory *Directory) DebugWrite(w io.Writer, prefix string) error {
	return directory.debugWrite(w, prefix)
}

type RegularInode struct {
	Mode             uint32
	Uid              uint32
	Gid              uint32
	MtimeNanoSeconds int32
	MtimeSeconds     int64
	Size             uint64
	Hash             hash.Hash
}

type RegularFile struct {
	Name        string
	InodeNumber uint64
	inode       *RegularInode // Keep private to avoid encoding/transmission.
}

func (file *RegularFile) Inode() *RegularInode {
	return file.inode
}

func (file *RegularFile) SetInode(inode *RegularInode) {
	file.inode = inode
}

func (file *RegularFile) String() string {
	return file.Name
}

func (file *RegularFile) DebugWrite(w io.Writer, prefix string) error {
	return file.debugWrite(w, prefix)
}

type SymlinkInode struct {
	Uid     uint32
	Gid     uint32
	Symlink string
}

type Symlink struct {
	Name        string
	InodeNumber uint64
	inode       *SymlinkInode // Keep private to avoid encoding/transmission.
}

func (symlink *Symlink) Inode() *SymlinkInode {
	return symlink.inode
}

func (symlink *Symlink) SetInode(inode *SymlinkInode) {
	symlink.inode = inode
}

func (symlink *Symlink) DebugWrite(w io.Writer, prefix string) error {
	return symlink.debugWrite(w, prefix)
}

type Inode struct {
	Mode             uint32
	Uid              uint32
	Gid              uint32
	MtimeNanoSeconds int32
	MtimeSeconds     int64
	Rdev             uint64
}

type File struct {
	Name        string
	InodeNumber uint64
	inode       *Inode // Keep private to avoid encoding/transmission.
}

func (file *File) Inode() *Inode {
	return file.inode
}

func (file *File) SetInode(inode *Inode) {
	file.inode = inode
}

func (file *File) String() string {
	return file.Name
}

func (file *File) DebugWrite(w io.Writer, prefix string) error {
	return file.debugWrite(w, prefix)
}

func CompareFileSystems(left, right *FileSystem, logWriter io.Writer) bool {
	return compareFileSystems(left, right, logWriter)
}

func CompareDirectories(left, right *Directory, logWriter io.Writer) bool {
	return compareDirectories(left, right, logWriter)
}

func CompareRegularFiles(left, right *RegularFile, logWriter io.Writer) bool {
	return compareRegularFiles(left, right, logWriter)
}

func CompareRegularInodes(left, right *RegularInode, logWriter io.Writer) bool {
	return compareRegularInodes(left, right, logWriter)
}

func CompareSymlinks(left, right *Symlink, logWriter io.Writer) bool {
	return compareSymlinks(left, right, logWriter)
}

func CompareSymlinkInodes(left, right *SymlinkInode, logWriter io.Writer) bool {
	return compareSymlinkInodes(left, right, logWriter)
}

func CompareFiles(left, right *File, logWriter io.Writer) bool {
	return compareFiles(left, right, logWriter)
}

func CompareInodes(left, right *Inode, logWriter io.Writer) bool {
	return compareInodes(left, right, logWriter)
}
