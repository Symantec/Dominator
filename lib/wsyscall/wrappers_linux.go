package wsyscall

import "syscall"

func convertStat(dest *Stat_t, source *syscall.Stat_t) {
	dest.Dev = source.Dev
	dest.Ino = source.Ino
	dest.Nlink = source.Nlink
	dest.Mode = source.Mode
	dest.Uid = source.Uid
	dest.Gid = source.Gid
	dest.Rdev = source.Rdev
	dest.Size = source.Size
	dest.Blksize = source.Blksize
	dest.Blocks = source.Blocks
	dest.Atim = source.Atim
	dest.Mtim = source.Mtim
	dest.Ctim = source.Ctim
}

func setAllUid(uid int) error {
	return syscall.Setresuid(uid, uid, uid)
}

func setAllGid(gid int) error {
	return syscall.Setresgid(gid, gid, gid)
}
