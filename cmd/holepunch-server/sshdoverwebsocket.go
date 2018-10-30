package main

import (
	"fmt"
	"github.com/function61/gokit/logger"
	"github.com/function61/holepunch-server/pkg/holepunchsshserver"
	"github.com/function61/holepunch-server/pkg/tcpkeepalive"
	"github.com/function61/holepunch-server/pkg/wsconnadapter"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"net"
	"net/http"
	"strings"
)

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func RegisterSshdOverWebsocket(mux *http.ServeMux, conf *ssh.ServerConfig) {
	log := logger.New("sshd-over-websocket")

	mux.HandleFunc("/_ssh", func(w http.ResponseWriter, r *http.Request) {
		upgrade := r.Header.Get("Upgrade")

		if strings.ToLower(upgrade) == "websocket" {
			wsConn, err := websocketUpgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Error(fmt.Sprintf("failure upgrading: %s", err.Error()))
				return
			}

			if err := tcpkeepalive.Enable(wsConn.UnderlyingConn().(*net.TCPConn), tcpkeepalive.DefaultDuration); err != nil {
				log.Error(fmt.Sprintf("tcpkeepalive: %s", err.Error()))
			}

			log.Info("Handing WS conn to SSH holepunchsshserver")

			holepunchsshserver.ServeConn(wsconnadapter.New(wsConn), conf)
		} else {
			log.Error("Someone called SSH endpoint without Websocket semantics")
			http.NotFound(w, r)
		}
	})
}
