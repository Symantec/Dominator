package hash

func (h Hash) marshalText() ([]byte, error) {
	retval := make([]byte, 0, 2*len(h))
	for _, byteVal := range h {
		retval = append(retval, formatNibble(byteVal>>4))
		retval = append(retval, formatNibble(byteVal&0xf))
	}
	return retval, nil
}

func formatNibble(nibble byte) byte {
	if nibble < 10 {
		return '0' + nibble
	}
	return 'a' + nibble - 10
}
