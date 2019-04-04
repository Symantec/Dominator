package wsyscall

import "syscall"

func fallocate(fd int, mode uint32, off int64, len int64) error {
	return syscall.ENOTSUP
}

func lstat(path string, statbuf *Stat_t) error {
	return syscall.ENOTSUP
}

func mount(source string, target string, fstype string, flags uintptr,
	data string) error {
	return syscall.ENOTSUP
}

func getrusage(who int, rusage *Rusage) error {
	return syscall.ENOTSUP
}

func setAllGid(gid int) error {
	return syscall.ENOTSUP
}

func setAllUid(uid int) error {
	return syscall.ENOTSUP
}

func setNetNamespace(namespaceFd int) error {
	return syscall.ENOTSUP
}

func unshareNetNamespace() (int, int, error) {
	return -1, -1, syscall.ENOTSUP
}

func unshareMountNamespace() error {
	return syscall.ENOTSUP
}
