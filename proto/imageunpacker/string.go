package imageunpacker

func (status StreamStatus) string() string {
	switch status {
	case StatusStreamIdle:
		return "idle"
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
