package main

import (
	"context"
	"log"
	"net"

	"github.com/function61/gokit/log/logex"
	"github.com/function61/gokit/sync/taskrunner"
	"github.com/function61/holepunch-server/pkg/holepunchsshserver"
	"golang.org/x/crypto/ssh"
)

func serveSshdOnTCP(
	ctx context.Context,
	addr string,
	conf *ssh.ServerConfig,
	logger *log.Logger,
) error {
	tcpListener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	logex.Levels(logger).Info.Printf("Listening on %s", addr)

	tasks := taskrunner.New(ctx, logger)

	tasks.Start("listener "+addr, func(ctx context.Context) error {
		for {
			tcpConn, err := tcpListener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return nil // expected error
				default:
					return err // unexpected error
				}
			}

			go holepunchsshserver.ServeConn(tcpConn, conf, logger)
		}
	})

	tasks.Start("listenercloser", func(ctx context.Context) error {
		<-ctx.Done()
		return tcpListener.Close()
	})

	return tasks.Wait()
}
