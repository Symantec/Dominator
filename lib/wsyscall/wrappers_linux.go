package wsyscall

import (
	"runtime"
	"syscall"
)

func convertStat(dest *Stat_t, source *syscall.Stat_t) {
	dest.Dev = source.Dev
	dest.Ino = source.Ino
	dest.Nlink = uint64(source.Nlink)
	dest.Mode = source.Mode
	dest.Uid = source.Uid
	dest.Gid = source.Gid
	dest.Rdev = source.Rdev
	dest.Size = source.Size
	dest.Blksize = int64(source.Blksize)
	dest.Blocks = source.Blocks
	dest.Atim = source.Atim
	dest.Mtim = source.Mtim
	dest.Ctim = source.Ctim
}

func fallocate(fd int, mode uint32, off int64, len int64) error {
	return syscall.Fallocate(fd, mode, off, len)
}

func mount(source string, target string, fstype string, flags uintptr,
	data string) error {
	var linuxFlags uintptr
	if flags&MS_BIND != 0 {
		linuxFlags |= syscall.MS_BIND
	}
	return syscall.Mount(source, target, fstype, linuxFlags, data)
}

func getrusage(who int, rusage *Rusage) error {
	switch who {
	case RUSAGE_CHILDREN:
		who = syscall.RUSAGE_CHILDREN
	case RUSAGE_SELF:
		who = syscall.RUSAGE_SELF
	case RUSAGE_THREAD:
		who = syscall.RUSAGE_THREAD
	default:
		return syscall.ENOTSUP
	}
	var syscallRusage syscall.Rusage
	if err := syscall.Getrusage(who, &syscallRusage); err != nil {
		return err
	}
	rusage.Utime.Sec = int64(syscallRusage.Utime.Sec)
	rusage.Utime.Usec = int64(syscallRusage.Utime.Usec)
	rusage.Stime.Sec = int64(syscallRusage.Stime.Sec)
	rusage.Stime.Usec = int64(syscallRusage.Stime.Usec)
	rusage.Maxrss = int64(syscallRusage.Maxrss)
	rusage.Ixrss = int64(syscallRusage.Ixrss)
	rusage.Idrss = int64(syscallRusage.Idrss)
	rusage.Minflt = int64(syscallRusage.Minflt)
	rusage.Majflt = int64(syscallRusage.Majflt)
	rusage.Nswap = int64(syscallRusage.Nswap)
	rusage.Inblock = int64(syscallRusage.Inblock)
	rusage.Oublock = int64(syscallRusage.Oublock)
	rusage.Msgsnd = int64(syscallRusage.Msgsnd)
	rusage.Msgrcv = int64(syscallRusage.Msgrcv)
	rusage.Nsignals = int64(syscallRusage.Nsignals)
	rusage.Nvcsw = int64(syscallRusage.Nvcsw)
	rusage.Nivcsw = int64(syscallRusage.Nivcsw)
	return nil
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
