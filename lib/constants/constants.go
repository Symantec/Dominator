package constants

const (
	SubPortNumber         = 6969
	DomPortNumber         = 6970
	ImageServerPortNumber = 6971

	DefaultNetworkSpeedPercent = 10
)

var ScanExcludeList = []string{
	"/tmp/.*",
	"/var/log/.*",
	"/var/mail/.*",
	"/var/spool/.*",
	"/var/tmp/.*",
}
