package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/function61/gokit/dynversion"
	"github.com/function61/gokit/envvar"
	"github.com/function61/gokit/httputils"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/taskrunner"
	"github.com/function61/holepunch-server/pkg/holepunchsshserver"
	"github.com/function61/holepunch-server/pkg/reverseproxy"
	"github.com/function61/holepunch-server/pkg/sshserverportforward"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func main() {
	app := &cobra.Command{
		Use:     os.Args[0],
		Short:   "holepunch-server",
		Version: dynversion.Version,
	}

	app.AddCommand(serverEntry())

	exitIfError(app.Execute())
}

func serverEntry() *cobra.Command {
	sshdOverWebsocket := false
	sshdOverTcp := ""
	reverseProxy := false

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start server",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			rootLogger := logex.StandardLogger()

			exitIfError(server(
				ossignal.InterruptOrTerminateBackgroundCtx(rootLogger),
				sshdOverWebsocket,
				sshdOverTcp,
				reverseProxy,
				rootLogger,
			))
		},
	}

	cmd.Flags().BoolVarP(&sshdOverWebsocket, "sshd-websocket", "", sshdOverWebsocket, "Serve holepunch-SSHD over WS")
	cmd.Flags().StringVarP(&sshdOverTcp, "sshd-tcp", "", sshdOverTcp, "Serve holepunch-SSHD over TCP, specify e.g. 0.0.0.0:22")
	cmd.Flags().BoolVarP(&reverseProxy, "http-reverse-proxy", "", reverseProxy, "Enable holepunch HTTP reverse proxy")

	return cmd
}

func server(
	ctx context.Context,
	sshdOverWebsocket bool,
	sshdOverTcp string,
	reverseProxy bool,
	logger *log.Logger,
) error {
	sshserverportforward.SetLogger(logex.Prefix("sshd-portforward", logger))

	logl := logex.Levels(logger)

	defer logl.Info.Println("Stopped")

	tasks := taskrunner.New(ctx, logger)

	logl.Info.Printf("holepunch-server %s starting", dynversion.Version)

	if sshdOverTcp != "" {
		sshConf, err := sshConfig()
		if err != nil {
			return err
		}

		tasks.Start("tcp-sshd", func(ctx context.Context, taskName string) error {
			return serveSshdOnTCP(
				ctx,
				sshdOverTcp,
				sshConf,
				logex.Prefix(taskName, logger))
		})
	}

	mux := http.NewServeMux()

	if sshdOverWebsocket {
		sshConf, err := sshConfig()
		if err != nil {
			return err
		}

		RegisterSshdOverWebsocket(
			mux,
			sshConf,
			logex.Prefix("ws", logger))
	}

	if reverseProxy {
		reverseproxy.Register(mux, logex.Prefix("reverseproxy", logger))
	}

	// only need HTTP if these services are enabled
	if sshdOverWebsocket || reverseProxy {
		tasks.Start("httpserver", func(ctx context.Context, taskName string) error {
			return serveHttp(ctx, mux, logex.Prefix(taskName, logger))
		})
	}

	return tasks.Wait()
}

func sshConfig() (*ssh.ServerConfig, error) {
	hostPrivateKey, err := envvar.RequiredFromBase64Encoded("SSH_HOSTKEY")
	if err != nil {
		return nil, err
	}

	clientPubKey, err := envvar.Required("CLIENT_PUBKEY")
	if err != nil {
		return nil, err
	}

	conf, err := holepunchsshserver.DefaultConfig(hostPrivateKey, clientPubKey)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func serveHttp(ctx context.Context, handler http.Handler, logger *log.Logger) error {
	srv := &http.Server{
		Addr:    ":80",
		Handler: handler,
	}

	tasks := taskrunner.New(ctx, logger)

	tasks.Start("listener "+srv.Addr, func(_ context.Context, _ string) error {
		return httputils.RemoveGracefulServerClosedError(srv.ListenAndServe())
	})

	tasks.Start("listenershutdowner", httputils.ServerShutdownTask(srv))

	return tasks.Wait()
}

func exitIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
