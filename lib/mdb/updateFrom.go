package mdb

func (dest *Machine) updateFrom(source Machine) {
	if dest.Hostname != source.Hostname {
		return
	}
	if source.IpAddress != "" {
		dest.IpAddress = source.IpAddress
	}
	if source.RequiredImage != "" {
		dest.RequiredImage = source.RequiredImage
		dest.DisableUpdates = source.DisableUpdates
	}
	if source.PlannedImage != "" {
		dest.PlannedImage = source.PlannedImage
	}
	if source.OwnerGroup != "" {
		dest.OwnerGroup = source.OwnerGroup
	}
	if source.AwsMetadata != nil {
		if dest.AwsMetadata == nil {
			dest.AwsMetadata = source.AwsMetadata
		} else if !compareAwsMetadata(dest.AwsMetadata, source.AwsMetadata) {
			dest.AwsMetadata = source.AwsMetadata
		}
	}
}

func compareAwsMetadata(left, right *AwsMetadata) bool {
	if left.InstanceId != right.InstanceId {
		return false
	}
	if len(left.Tags) != len(right.Tags) {
		return false
	}
	for key, value := range left.Tags {
		if right.Tags[key] != value {
			return false
		}
	}
	return true
}
