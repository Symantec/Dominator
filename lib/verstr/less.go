package verstr

import (
	"strconv"
	"strings"
)

func less(left, right string) bool {
	leftFields := strings.Split(left, ".")
	for index, rightField := range strings.Split(right, ".") {
		if index >= len(leftFields) {
			return true
		}
		leftField := leftFields[index]
		if rightVal, err := strconv.ParseUint(rightField, 10, 64); err == nil {
			if leftVal, err := strconv.ParseUint(
				leftField, 10, 64); err == nil {
				if leftVal < rightVal {
					return true
				} else if leftVal > rightVal {
					return false
				}
				continue
			}
		}
		if leftField < rightField {
			return true
		} else if leftField > rightField {
			return false
		}
	}
	return false
}
