package objectcache

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
)

func filenameToHash(fileName string) (hash.Hash, error) {
	var hash hash.Hash
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
			hash[index] = nibble | prev_nibble<<4
			index++
			prev_nibble = 16
		} else {
			prev_nibble = nibble
		}
	}
	return hash, nil
}

func hashToFilename(hash hash.Hash) string {
	return fmt.Sprintf("%x/%x/%x", hash[0], hash[1], hash[2:])
}
