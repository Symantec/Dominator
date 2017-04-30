package amipublisher

import (
	"encoding/json"
)

func (v InstanceResult) marshalJSON() ([]byte, error) {
	errString := ""
	if v.Error != nil {
		errString = v.Error.Error()
	}
	val := struct {
		AccountName string
		Region      string
		InstanceId  string `json:",omitempty"`
		PrivateIp   string `json:",omitempty"`
		Error       string `json:",omitempty"`
	}{v.AccountName, v.Region, v.InstanceId, v.PrivateIp, errString}
	return json.Marshal(val)
}

func (v TargetResult) marshalJSON() ([]byte, error) {
	errString := ""
	if v.Error != nil {
		errString = v.Error.Error()
	}
	val := struct {
		AccountName    string
		Region         string
		SharedFrom     string `json:",omitempty"`
		SnapshotId     string `json:",omitempty"`
		S3Bucket       string `json:",omitempty"`
		S3ManifestFile string `json:",omitempty"`
		AmiId          string `json:",omitempty"`
		Size           uint   `json:",omitempty"`
		Error          string `json:",omitempty"`
	}{v.AccountName, v.Region, v.SharedFrom, v.SnapshotId, v.S3Bucket,
		v.S3ManifestFile, v.AmiId, v.Size, errString}
	return json.Marshal(val)
}
