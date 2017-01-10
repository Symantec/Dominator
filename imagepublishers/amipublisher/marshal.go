package amipublisher

import (
	"encoding/json"
)

func (v TargetResult) marshalJSON() ([]byte, error) {
	errString := ""
	if v.Error != nil {
		errString = v.Error.Error()
	}
	val := struct {
		AccountName string
		Region      string
		SnapshotId  string `json:",omitempty"`
		AmiId       string `json:",omitempty"`
		Size        uint   `json:",omitempty"`
		Error       string `json:",omitempty"`
	}{v.AccountName, v.Region, v.SnapshotId, v.AmiId, v.Size, errString}
	return json.Marshal(val)
}
