package srpc

func (conn *Conn) close() error {
	err := conn.Flush()
	if conn.parent != nil {
		conn.parent.callLock.Unlock()
	}
	return err
}
