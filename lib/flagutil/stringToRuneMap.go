package flagutil

import (
	"errors"
	"sort"
	"strings"
)

func (m *StringToRuneMap) String() string {
	keys := make([]string, 0, len(*m))
	for key := range *m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	retval := `"`
	for index, key := range keys {
		if index != 0 {
			retval += ","
		}
		retval += key + ":" + string((*m)[key])
	}
	return retval + `"`
}

func (m *StringToRuneMap) Set(value string) error {
	newMap := make(map[string]rune)
	for _, entry := range strings.Split(value, ",") {
		fields := strings.Split(entry, ":")
		if len(fields) != 2 {
			return errors.New("invalid entry: " + entry)
		}
		if len(fields[1]) != 1 {
			return errors.New("invalid filetype: " + fields[1])
		}
		newMap[fields[0]] = rune(fields[1][0])
	}
	*m = newMap
	return nil
}
