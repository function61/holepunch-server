package main

import (
	"github.com/function61/gokit/stopper"
	"github.com/function61/gokit/tcpkeepalive"
	"github.com/function61/holepunch-server/pkg/holepunchsshserver"
	"golang.org/x/crypto/ssh"
	"net"
)

func serveSshdOnTCP(addr string, conf *ssh.ServerConfig, stop *stopper.Stopper) {
	defer stop.Done()

	tcpListener, err := net.Listen("tcp", addr)
	if err != nil {
		sshdOverTcpLog.Error.Printf("Failed to listen on %s (%s)", addr, err)
		return
	}

	go func() {
		<-stop.Signal
		tcpListener.Close()
	}()

	sshdOverTcpLog.Info.Printf("Listening on %s", addr)

	for {
		tcpConn, err := tcpListener.Accept()
		if err != nil {
			sshdOverTcpLog.Error.Printf("Accept() failed: %s", err)
			break
		}

		if err := tcpkeepalive.Enable(tcpConn.(*net.TCPConn), tcpkeepalive.DefaultDuration); err != nil {
			sshdOverTcpLog.Error.Printf("tcpkeepalive: %s", err.Error())
		}

		go holepunchsshserver.ServeConn(tcpConn, conf, sshdServerLog)
	}
}
