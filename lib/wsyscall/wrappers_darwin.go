package wsyscall

import (
	"os"
	"syscall"
)

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

func getrusage(who int, rusage *Rusage) error {
	switch who {
	case RUSAGE_CHILDREN:
		who = syscall.RUSAGE_CHILDREN
	case RUSAGE_SELF:
		who = syscall.RUSAGE_SELF
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
	rusage.Maxrss = int64(syscallRusage.Maxrss) >> 10
	rusage.Ixrss = int64(syscallRusage.Ixrss) >> 10
	rusage.Idrss = int64(syscallRusage.Idrss) >> 10
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

func fallocate(fd int, mode uint32, off int64, len int64) error {
	return syscall.ENOTSUP
}

func ioctl(fd int, request, argp uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), request,
		argp)
	if errno != 0 {
		return os.NewSyscallError("ioctl", errno)
	}
	return nil
}

func lstat(path string, statbuf *Stat_t) error {
	var rawStatbuf syscall.Stat_t
	if err := syscall.Lstat(path, &rawStatbuf); err != nil {
		return err
	}
	convertStat(statbuf, &rawStatbuf)
	return nil
}

func mount(source string, target string, fstype string, flags uintptr,
	data string) error {
	return syscall.ENOTSUP
}

func setAllGid(gid int) error {
	return syscall.Setregid(gid, gid)
}

func setAllUid(uid int) error {
	return syscall.Setreuid(uid, uid)
}

func setNetNamespace(namespaceFd int) error {
	return syscall.ENOTSUP
}

func stat(path string, statbuf *Stat_t) error {
	var rawStatbuf syscall.Stat_t
	if err := syscall.Stat(path, &rawStatbuf); err != nil {
		return err
	}
	convertStat(statbuf, &rawStatbuf)
	return nil
}

func unshareNetNamespace() (int, int, error) {
	return -1, -1, syscall.ENOTSUP
}

func unshareMountNamespace() error {
	return syscall.ENOTSUP
}
