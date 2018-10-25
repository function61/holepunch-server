package main

import (
	"fmt"
	"github.com/function61/gokit/envvar"
	"github.com/function61/gokit/logger"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/stopper"
	"github.com/function61/holepunch-server/pkg/holepunchsshserver"
	"github.com/function61/holepunch-server/pkg/reverseproxy"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"net/http"
	"os"
)

var version = "dev" // replaced dynamically at build time

func serverEntry() *cobra.Command {
	log := logger.New("holepunch-server")

	sshdOverWebsocket := false
	sshdOverTcp := ""
	reverseProxy := false

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start server",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			defer log.Info("Stopped")

			log.Info(fmt.Sprintf("holepunch-server %s starting", version))

			workers := stopper.NewManager()

			if sshdOverTcp != "" {
				go serveSshdOnTCP(sshdOverTcp, sshConfig(), workers.Stopper())
			}

			if sshdOverWebsocket {
				RegisterSshdOverWebsocket(http.DefaultServeMux, sshConfig())
			}

			if reverseProxy {
				reverseproxy.Register(http.DefaultServeMux)
			}

			// only need HTTP if these services are enabled
			if sshdOverWebsocket || reverseProxy {
				go serveHttp(workers.Stopper())
			}

			log.Info(fmt.Sprintf("Got %s; stopping", ossignal.WaitForInterruptOrTerminate()))

			workers.StopAllWorkersAndWait()
		},
	}

	cmd.Flags().BoolVarP(&sshdOverWebsocket, "sshd-websocket", "", sshdOverWebsocket, "Serve holepunch-SSHD over WS")
	cmd.Flags().StringVarP(&sshdOverTcp, "sshd-tcp", "", sshdOverTcp, "Serve holepunch-SSHD over TCP, specify e.g. 0.0.0.0:22")
	cmd.Flags().BoolVarP(&reverseProxy, "http-reverse-proxy", "", reverseProxy, "Enable holepunch HTTP reverse proxy")

	return cmd
}

func main() {
	app := &cobra.Command{
		Use:     os.Args[0],
		Short:   "holepunch-server",
		Version: version,
	}

	app.AddCommand(serverEntry())

	if err := app.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func sshConfig() *ssh.ServerConfig {
	hostPrivateKey, err := envvar.GetFromBase64Encoded("SSH_HOSTKEY")
	if err != nil {
		panic(err)
	}

	clientPubKey, err := envvar.Get("CLIENT_PUBKEY")
	if err != nil {
		panic(err)
	}

	conf, err := holepunchsshserver.DefaultConfig(hostPrivateKey, clientPubKey)
	if err != nil {
		panic(err)
	}

	return conf
}

func serveHttp(stop *stopper.Stopper) {
	defer stop.Done()

	srv := &http.Server{
		Addr: ":80",
	}

	go func() {
		<-stop.Signal
		srv.Shutdown(nil)
	}()

	srv.ListenAndServe()
}
