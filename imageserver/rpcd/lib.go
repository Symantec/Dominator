package rpcd

import (
	"errors"
)

func (t *srpcType) checkMutability() error {
	if t.replicationMaster != "" {
		return errors.New(replicationMessage + t.replicationMaster)
	}
	return nil
}
