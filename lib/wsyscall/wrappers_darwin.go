package wsyscall

import "syscall"

func convertStat(dest *Stat_t, source *syscall.Stat_t) {
	dest.Dev = uint64(source.Dev)
	dest.Ino = source.Ino
	dest.Nlink = uint64(source.Nlink)
	dest.Mode = uint32(source.Mode)
	dest.Uid = source.Uid
	dest.Gid = source.Gid
	dest.Rdev = uint64(source.Rdev)
	dest.Size = source.Size
	dest.Blksize = int64(source.Blksize)
	dest.Blocks = source.Blocks
	dest.Atim = source.Atimespec
	dest.Mtim = source.Mtimespec
	dest.Ctim = source.Ctimespec
}

func mount(source string, target string, fstype string, flags uintptr,
	data string) error {
	return syscall.ENOTSUP
}

func getrusage(who int, rusage *syscall.Rusage) error {
	switch who {
	case RUSAGE_CHILDREN:
		return syscall.Getrusage(syscall.RUSAGE_CHILDREN, rusage)
	case RUSAGE_SELF:
		return syscall.Getrusage(syscall.RUSAGE_SELF, rusage)
	default:
		return syscall.ENOTSUP
	}
}

func setAllGid(gid int) error {
	return syscall.Setregid(gid, gid)
}

func setAllUid(uid int) error {
	return syscall.Setreuid(uid, uid)
}

func unshareMountNamespace() error {
	return syscall.ENOTSUP
}
