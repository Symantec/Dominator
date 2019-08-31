package installer

const (
	FileSystemTypeExt4 = 0
)

type FileSystemType uint

type Partition struct {
	FileSystemType   FileSystemType `json:",omitempty"`
	MountPoint       string         `json:",omitempty"`
	MinimumFreeBytes uint64         `json:",omitempty"`
}

type StorageLayout struct {
	BootDriveLayout          []Partition `json:",omitempty"`
	ExtraMountPointsBasename string      `json:",omitempty"`
	Encrypt                  bool        `json:",omitempty"`
	UseKexec                 bool        `json:",omitempty"`
}
