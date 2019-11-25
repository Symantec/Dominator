package flagutil

import (
	"bytes"
	"strings"
)

func (sl *SizeList) String() string {
	buffer := &bytes.Buffer{}
	buffer.WriteString(`"`)
	for index, size := range *sl {
		buffer.WriteString(size.String())
		if index < len(*sl)-1 {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString(`"`)
	return buffer.String()
}

func (sl *SizeList) Set(value string) error {
	newList := make(SizeList, 0)
	if value != "" {
		sizeStrings := strings.Split(value, ",")
		for _, sizeString := range sizeStrings {
			var size Size
			if err := size.Set(sizeString); err != nil {
				return err
			}
			newList = append(newList, size)
		}
	}
	*sl = newList
	return nil
}
