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
}
