package imageunpacker

func (status StreamStatus) string() string {
	switch status {
	case StatusStreamNoDevice:
		return "no device"
	case StatusStreamNotMounted:
		return "not mounted"
	case StatusStreamMounted:
		return "mounted"
	case StatusStreamScanning:
		return "scanning"
	case StatusStreamScanned:
		return "scanned"
	case StatusStreamFetching:
		return "fetching"
	case StatusStreamUpdating:
		return "updating"
	case StatusStreamPreparing:
		return "preparing"
	default:
		return "UNKNOWN"
	}
}
