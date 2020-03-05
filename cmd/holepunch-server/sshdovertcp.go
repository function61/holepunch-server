package main

import (
	"context"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/taskrunner"
	"github.com/function61/holepunch-server/pkg/holepunchsshserver"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
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

	tasks.Start("listener "+addr, func(ctx context.Context, _ string) error {
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

	tasks.Start("listenercloser", func(ctx context.Context, _ string) error {
		<-ctx.Done()
		return tcpListener.Close()
	})

	return tasks.Wait()
}
