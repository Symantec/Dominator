package srpc

import "io"

func (conn *Conn) close() error {
	err := conn.Flush()
	if conn.parent != nil {
		conn.parent.callLock.Unlock()
	}
	return err
}

func (conn *Conn) getAuthInformation() *AuthInformation {
	if conn.parent != nil {
		panic("cannot call GetAuthInformation() for client connection")
	}
	return &AuthInformation{conn.haveMethodAccess, conn.username}
}

func (conn *Conn) getCloseNotifier() <-chan error {
	closeChannel := make(chan error)
	go func() {
		for {
			buf := make([]byte, 1)
			if _, err := conn.Read(buf); err != nil {
				if err == io.EOF {
					err = nil
				}
				closeChannel <- err
				return
			}
		}
	}()
	return closeChannel
}

func (conn *Conn) getUsername() string {
	if conn.parent != nil {
		panic("cannot call GetUsername() for client connection")
	}
	return conn.username
}
