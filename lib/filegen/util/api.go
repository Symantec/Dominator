package util

import (
	"github.com/Symantec/Dominator/lib/filegen"
)

func LoadConfiguration(manager *filegen.Manager, filename string) error {
	return loadConfiguration(manager, filename)
}
