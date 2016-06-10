package setupclient

func SetupTls(ignoreMissingCerts bool) error {
	return setupTls(ignoreMissingCerts)
}
