package constants

const (
	SubPortNumber                = 6969
	DomPortNumber                = 6970
	ImageServerPortNumber        = 6971
	BasicFileGenServerPortNumber = 6972

	DefaultNetworkSpeedPercent = 10
)

var ScanExcludeList = []string{
	"/home/.*",
	"/tmp/.*",
	"/var/log/.*",
	"/var/mail/.*",
	"/var/spool/.*",
	"/var/tmp/.*",
}
