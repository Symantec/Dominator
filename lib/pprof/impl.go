package pprof

import (
	"os"
	"runtime/pprof"
)

var (
	cpuProfile *os.File
)

func startCpuProfile(filename string) error {
	if cpuProfile != nil {
		panic("CPU profiling already started")
	}
	if file, err := os.Create(filename); err != nil {
		return err
	} else {
		if err := pprof.StartCPUProfile(file); err != nil {
			file.Close()
			return err
		}
		cpuProfile = file
		return nil
	}
}

func stopCpuProfile() {
	if cpuProfile == nil {
		return
	}
	pprof.StopCPUProfile()
	cpuProfile.Close()
	cpuProfile = nil
}
