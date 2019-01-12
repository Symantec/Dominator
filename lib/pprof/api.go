package pprof

func StartCpuProfile(filename string) error {
	return startCpuProfile(filename)
}

func StopCpuProfile() {
	stopCpuProfile()
}
