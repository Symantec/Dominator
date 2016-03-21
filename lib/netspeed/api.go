package netspeed

func GetSpeedToAddress(address string) (uint64, bool) {
	return getSpeedToAddress(address)
}

func GetSpeedToHost(hostname string) (uint64, bool) {
	return getSpeedToHost(hostname)
}
