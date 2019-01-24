package objectcache

import (
	"fmt"

	"github.com/Symantec/Dominator/lib/hash"
)

func filenameToHash(fileName string) (hash.Hash, error) {
	var hashVal hash.Hash
	var prev_nibble byte = 16
	index := 0
	for _, char := range fileName {
		var nibble byte = 16
		if char >= '0' && char <= '9' {
			nibble = byte(char) - '0'
		} else if char >= 'a' && char <= 'f' {
			nibble = byte(char) - 'a' + 10
		} else {
			continue // Ignore everything else. Treat them as separators.
		}
		if prev_nibble < 16 {
			if index >= len(hashVal) {
				return hashVal, fmt.Errorf("filename too long: %s", fileName)
			}
			hashVal[index] = nibble | prev_nibble<<4
			index++
			prev_nibble = 16
		} else {
			prev_nibble = nibble
		}
	}
	return hashVal, nil
}

func hashToFilename(hashVal hash.Hash) string {
	return fmt.Sprintf("%02x/%02x/%0x", hashVal[0], hashVal[1], hashVal[2:])
}
