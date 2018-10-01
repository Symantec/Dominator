package fsstorer

import "fmt"

func (ip IP) string() string {
	return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
}
