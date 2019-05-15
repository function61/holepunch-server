package main

import (
	"github.com/function61/gokit/tcpkeepalive"
	"github.com/function61/holepunch-server/pkg/holepunchsshserver"
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
	mux.HandleFunc("/_ssh", func(w http.ResponseWriter, r *http.Request) {
		upgrade := r.Header.Get("Upgrade")

		if strings.ToLower(upgrade) == "websocket" {
			wsConn, err := websocketUpgrader.Upgrade(w, r, nil)
			if err != nil {
				sshdOverWsLog.Error.Printf("failure upgrading: %s", err.Error())
				return
			}

			if err := tcpkeepalive.Enable(wsConn.UnderlyingConn().(*net.TCPConn), tcpkeepalive.DefaultDuration); err != nil {
				sshdOverWsLog.Error.Printf("tcpkeepalive: %s", err.Error())
			}

			sshdOverWsLog.Info.Println("handoff to holepunchsshserver")

			holepunchsshserver.ServeConn(wsconnadapter.New(wsConn), conf, sshdServerLog)
		} else {
			sshdOverWsLog.Error.Println("SSH endpoint called without Websocket semantics")
			http.NotFound(w, r)
		}
	})
}
