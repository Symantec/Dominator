package meminfo

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var procMeminfo string = "/proc/meminfo"

func getMemInfo() (*MemInfo, error) {
	file, err := os.Open(procMeminfo)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	memInfo := new(MemInfo)
	for scanner.Scan() {
		if err := memInfo.processMeminfoLine(scanner.Text()); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return memInfo, nil
}

func (m *MemInfo) processMeminfoLine(line string) error {
	splitLine := strings.SplitN(line, ":", 2)
	if len(splitLine) != 2 {
		return nil
	}
	meminfoName := splitLine[0]
	meminfoDataString := strings.TrimSpace(splitLine[1])
	var ptr *uint64
	switch meminfoName {
	case "MemAvailable":
		ptr = &m.Available
	case "MemFree":
		ptr = &m.Free
	case "MemTotal":
		ptr = &m.Total
	default:
		return nil
	}
	var meminfoData uint64
	var meminfoUnit string
	fmt.Sscanf(meminfoDataString, "%d %s", &meminfoData, &meminfoUnit)
	if meminfoUnit != "kB" {
		return fmt.Errorf("unknown unit: %s for: %s", meminfoUnit, meminfoName)
	}
	*ptr = meminfoData * 1024
	return nil
}
