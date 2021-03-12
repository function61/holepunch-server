package main

import (
	"log"
	"net/http"

	"github.com/function61/gokit/log/logex"
	"github.com/function61/holepunch-server/pkg/holepunchsshserver"
	"github.com/function61/holepunch-server/pkg/wsconnadapter"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func RegisterSshdOverWebsocket(mux *http.ServeMux, conf *ssh.ServerConfig, logger *log.Logger) {
	logl := logex.Levels(logger)

	sshdLogger := logex.Prefix("sshd", logger)

	mux.HandleFunc("/_ssh", func(w http.ResponseWriter, r *http.Request) {
		// checks for proper "Upgrade: websocket" header
		wsConn, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			logl.Error.Printf("failure upgrading: %s", err.Error())
			return
		}

		logl.Info.Println("handoff to holepunchsshserver")

		holepunchsshserver.ServeConn(wsconnadapter.New(wsConn), conf, sshdLogger)
	})
}
