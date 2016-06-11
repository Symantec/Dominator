package setupserver

func SetupTls() error {
	return setupTls(true)
}

func SetupTlsClientOnly() error {
	return setupTls(false)
}
