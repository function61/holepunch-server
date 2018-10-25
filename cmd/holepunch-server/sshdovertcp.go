package main

import (
	"fmt"
	"github.com/function61/gokit/logger"
	"github.com/function61/gokit/stopper"
	"github.com/function61/holepunch-server/pkg/holepunchsshserver"
	"golang.org/x/crypto/ssh"
	"net"
)

func serveSshdOnTCP(addr string, conf *ssh.ServerConfig, stop *stopper.Stopper) {
	defer stop.Done()

	log := logger.New("sshd-over-tcp")

	tcpListener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to listen on %s (%s)", addr, err))
		return
	}

	go func() {
		<-stop.Signal
		tcpListener.Close()
	}()

	log.Info(fmt.Sprintf("Listening on %s", addr))

	for {
		tcpConn, err := tcpListener.Accept()
		if err != nil {
			log.Error(fmt.Sprintf("Accept() failed: %s", err))
			break
		}

		go holepunchsshserver.ServeConn(tcpConn, conf)
	}
}
