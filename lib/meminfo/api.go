package meminfo

type MemInfo struct {
	Available uint64
	Free      uint64
	Total     uint64
}

func GetMemInfo() (*MemInfo, error) {
	return getMemInfo()
}
