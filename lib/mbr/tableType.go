package mbr

import (
	"fmt"
)

var tableTypeToString = map[TableType]string{
	TABLE_TYPE_AIX:   "aix",
	TABLE_TYPE_BSD:   "bsd",
	TABLE_TYPE_DVH:   "dvh",
	TABLE_TYPE_GPT:   "gpt",
	TABLE_TYPE_LOOP:  "loop",
	TABLE_TYPE_MAC:   "mac",
	TABLE_TYPE_MSDOS: "msdos",
	TABLE_TYPE_PC98:  "pc98",
	TABLE_TYPE_SUN:   "sun",
}

func (tableType TableType) lookupString() (string, error) {
	if label, ok := tableTypeToString[tableType]; !ok {
		return "", fmt.Errorf("unknown table type: %d", tableType)
	} else {
		return label, nil
	}
}

func (tt *TableType) set(value string) error {
	for tableType, name := range tableTypeToString {
		if value == name {
			*tt = tableType
			return nil
		}
	}
	return fmt.Errorf("unknown table type: %s", value)
}

func (tt TableType) string() string {
	if label, err := tt.lookupString(); err != nil {
		return err.Error()
	} else {
		return label
	}
}
