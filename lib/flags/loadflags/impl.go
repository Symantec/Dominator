package loadflags

import (
	"bufio"
	"errors"
	"flag"
	"os"
	"strings"
)

func loadFlags(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) < 1 {
			continue
		}
		if line[0] == '#' || line[0] == ';' {
			continue
		}
		splitLine := strings.SplitN(line, "=", 2)
		if len(splitLine) < 2 {
			return errors.New("bad line, cannot split name from value: " + line)
		}
		name := strings.TrimSpace(splitLine[0])
		if strings.Count(name, " ") != 0 {
			return errors.New("bad line, name has whitespace: " + line)
		}
		value := strings.TrimSpace(splitLine[1])
		if err := flag.CommandLine.Set(name, value); err != nil {
			return err
		}
	}
	return scanner.Err()
}
