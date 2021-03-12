package holepunchsshserver

import (
	"bytes"
	"errors"
	"log"
	"net"

	"github.com/function61/gokit/log/logex"
	"github.com/function61/holepunch-server/pkg/sshserverportforward"
	"golang.org/x/crypto/ssh"
)

func ServeConn(conn net.Conn, config *ssh.ServerConfig, logger *log.Logger) {
	logl := logex.Levels(logger)

	// Before use, a handshake must be performed on the incoming net.Conn.
	sshServerConn, newChannelRequests, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		logl.Error.Printf("Failed to handshake (%s)", err)
		return
	}

	logl.Info.Printf("Authorized user %s from %s (%s)",
		sshServerConn.User(),
		sshServerConn.RemoteAddr(),
		sshServerConn.ClientVersion())

	// handle portforwarding out-of-band requests, but discard all other
	// these are reverse forwards
	nonForwardReqs := sshserverportforward.ProcessPortForwardRequests(requests, sshServerConn)
	go ssh.DiscardRequests(nonForwardReqs)

	// these are normal forwards ("forward forwards")
	nonForwardChans := sshserverportforward.ProcessPortForwardNewChannelRequests(newChannelRequests)
	go sshserverportforward.RejectChannelRequests(nonForwardChans)
}

func DefaultConfig(hostPrivateKeyBytes []byte, clientPubKey string) (*ssh.ServerConfig, error) {
	config := &ssh.ServerConfig{
		PublicKeyCallback: keyAuthorizer(clientPubKey),
	}

	hostPrivateKey, err := ssh.ParsePrivateKey(hostPrivateKeyBytes)
	if err != nil {
		return nil, err
	}

	config.AddHostKey(hostPrivateKey)

	return config, nil
}

func keyAuthorizer(expectedClientKeyAuthorizedFormat string) func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
	return func(metadata ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		if metadata.User() != "hp" {
			return nil, errors.New("unknown username")
		}

		expectedClientKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(expectedClientKeyAuthorizedFormat))
		if err != nil {
			return nil, err
		}

		if !publicKeysEqual(key, expectedClientKey) {
			return nil, errors.New("client pubkey not authorized")
		}

		return nil, nil
	}
}

func publicKeysEqual(key1 ssh.PublicKey, key2 ssh.PublicKey) bool {
	return bytes.Equal(key1.Marshal(), key2.Marshal())
}
