package constants

const (
	SubPortNumber                = 6969
	DominatorPortNumber          = 6970
	ImageServerPortNumber        = 6971
	BasicFileGenServerPortNumber = 6972
	SimpleMdbServerPortNumber    = 6973
	ImageUnpackerPortNumber      = 6974
	ImaginatorPortNumber         = 6975
	HypervisorPortNumber         = 6976
	ClusterManagerPortNumber     = 6977

	DefaultCpuPercent          = 50
	DefaultNetworkSpeedPercent = 10
	DefaultScanSpeedPercent    = 2

	AssignedOIDBase        = "1.3.6.1.4.1.9586.100.7"
	PermittedMethodListOID = AssignedOIDBase + ".1"
)

var RequiredPaths = map[string]rune{
	"/etc":        'd',
	"/etc/passwd": 'f',
	"/usr":        'd',
	"/usr/bin":    'd',
}

var ScanExcludeList = []string{
	"/home/.*",
	"/tmp/.*",
	"/var/log/.*",
	"/var/mail/.*",
	"/var/spool/.*",
	"/var/tmp/.*",
}
