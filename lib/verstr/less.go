package verstr

import (
	"strconv"
	"strings"
)

func less(left, right string) bool {
	rightFields := strings.Split(right, ".")
	for index, leftField := range strings.Split(left, ".") {
		if index >= len(rightFields) {
			return false
		}
		rightField := rightFields[index]
		if leftVal, err := strconv.ParseUint(leftField, 10, 64); err == nil {
			if rightVal, err := strconv.ParseUint(
				rightField, 10, 64); err == nil {
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
