package verstr

func less(left, right string) bool {
	leftIndex := 0
	rightIndex := 0
	for {
		if rightIndex >= len(right) {
			return false
		}
		if leftIndex >= len(left) {
			return true
		}
		leftRune := left[leftIndex]
		rightRune := right[rightIndex]
		if leftRune >= '0' && leftRune <= '9' &&
			rightRune >= '0' && rightRune <= '9' {
			var diff int64
			diff, leftIndex, rightIndex = compareNumstr(left, leftIndex,
				right, rightIndex)
			if diff < 0 {
				return true
			} else if diff > 0 {
				return false
			} else {
				continue
			}
		}
		if leftRune < rightRune {
			return true
		} else if leftRune > rightRune {
			return false
		}
		leftIndex++
		rightIndex++
	}
}

func compareNumstr(left string, leftIndex int, right string, rightIndex int) (
	int64, int, int) {
	var leftVal, rightVal int64
	for ; leftIndex < len(left); leftIndex++ {
		char := int64(left[leftIndex])
		if char >= '0' && char <= '9' {
			leftVal = leftVal*10 + char - '0'
		} else {
			break
		}
	}
	for ; rightIndex < len(right); rightIndex++ {
		char := int64(right[rightIndex])
		if char >= '0' && char <= '9' {
			rightVal = rightVal*10 + char - '0'
		} else {
			break
		}
	}
	return leftVal - rightVal, leftIndex, rightIndex
}
