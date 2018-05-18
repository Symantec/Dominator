package topology

import (
	"fmt"
	"os"
	"strings"
)

func splitPath(path string) []string {
	return strings.Split(path, string(os.PathSeparator))
}

func (t *Topology) findDirectory(dirname string) (*Directory, error) {
	directory := t.Root
	if dirname == "" {
		return directory, nil
	}
	for list := splitPath(dirname); len(list) > 0; {
		if subdir, ok := directory.nameToDirectory[list[0]]; !ok {
			return nil, fmt.Errorf("directory: %s not found", dirname)
		} else {
			directory = subdir
			list = list[1:]
		}
	}
	return directory, nil
}

func (t *Topology) listMachines(dirname string) ([]*Machine, error) {
	directory, err := t.findDirectory(dirname)
	if err != nil {
		return nil, err
	}
	return directory.listMachines(), nil
}

func (directory *Directory) listMachines() []*Machine {
	machines := directory.Machines
	for _, subdir := range directory.Directories {
		machines = append(machines, subdir.listMachines()...)
	}
	return machines
}
