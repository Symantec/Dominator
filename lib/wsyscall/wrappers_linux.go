package wsyscall

import (
	"runtime"
	"syscall"
)

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

func mount(source string, target string, fstype string, flags uintptr,
	data string) error {
	var linuxFlags uintptr
	if flags&MS_BIND != 0 {
		linuxFlags |= syscall.MS_BIND
	}
	return syscall.Mount(source, target, fstype, linuxFlags, data)
}

func setAllGid(gid int) error {
	return syscall.Setresgid(gid, gid, gid)
}

func setAllUid(uid int) error {
	return syscall.Setresuid(uid, uid, uid)
}

func unshareMountNamespace() error {
	// Pin goroutine to OS thread. This hack is required because
	// syscall.Unshare() operates on only one thread in the process, and Go
	// switches execution between threads randomly. Thus, the namespace can be
	// suddenly switched for running code. This is an aspect of Go that was not
	// well thought out.
	runtime.LockOSThread()
	if err := syscall.Unshare(syscall.CLONE_NEWNS); err != nil {
		return err
	}
	return syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, "")
}
