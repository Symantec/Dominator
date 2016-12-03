package hash

type Hash [64]byte

func (h Hash) MarshalText() ([]byte, error) {
	return h.marshalText()
}
