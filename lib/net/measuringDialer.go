package net

import (
	"time"
)

func newMeasuringDialer(dialer Dialer) *MeasuringDialer {
	return &MeasuringDialer{dialer: dialer}
}

func (d *MeasuringDialer) Dial(network, address string) (
	*MeasuringConnection, error) {
	startTime := time.Now()
	netConn, err := d.dialer.Dial(network, address)
	d.cumulativeDialTime += time.Since(startTime)
	if err != nil {
		return nil, err
	}
	return &MeasuringConnection{Conn: netConn}, nil
}

func (conn *MeasuringConnection) read(b []byte) (n int, err error) {
	startTime := time.Now()
	n, err = conn.Conn.Read(b)
	conn.cumulativeReadTime += time.Since(startTime)
	return
}

func (conn *MeasuringConnection) write(b []byte) (n int, err error) {
	startTime := time.Now()
	n, err = conn.Conn.Write(b)
	conn.cumulativeWriteTime += time.Since(startTime)
	return
}
