package tcpkeepalive

import (
	"net"
	"time"
)

const (
	DefaultDuration = 15 * time.Second
)

func Enable(tcpConn *net.TCPConn, duration time.Duration) error {
	if err := tcpConn.SetKeepAlivePeriod(duration); err != nil {
		return err
	}

	return tcpConn.SetKeepAlive(true)
}
