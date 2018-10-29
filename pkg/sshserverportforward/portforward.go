package sshserverportforward

import (
	"errors"
	"fmt"
	"github.com/function61/gokit/bidipipe"
	"github.com/function61/gokit/logger"
	"golang.org/x/crypto/ssh"
	"net"
	"strconv"
)

// Go's (as of version 1.11) SSH implements port forwarding for client side only. this
// implements port forwarding for server side in a pluggable manner (one function call only).
//
// currently only reverse tunnels are supported. PRs are welcome :)

var log = logger.New("sshd-portforward")

// this needs to be global (because TCP ports are global)
var fwdList = &forwardList{
	reverseCancellations: map[string]chan bool{},
}

var (
	errUnsupportedAddress = errors.New("unsupported address")
)

// returns a new channel that receives all non-portforwarding requests.
// if you don't do anything with them call "go ssh.DiscardRequests()"
func ProcessPortForwardRequests(requests <-chan *ssh.Request, serverConn *ssh.ServerConn) <-chan *ssh.Request {
	nonForwardRequests := make(chan *ssh.Request, 1)

	go func() {
		for req := range requests {
			switch req.Type {
			case "tcpip-forward":
				processTcpipForwardReq(req, serverConn, fwdList)
			case "cancel-tcpip-forward":
				processTcpipCancelForwardReq(req, fwdList)
			default:
				nonForwardRequests <- req
			}
		}
	}()

	return nonForwardRequests
}

func processTcpipForwardReq(req *ssh.Request, serverConn *ssh.ServerConn, fwdList *forwardList) {
	var forwardingDetails channelForwardMsg
	if err := ssh.Unmarshal(req.Payload, &forwardingDetails); err != nil {
		log.Error(err.Error())
		req.Reply(false, nil)
		return
	}

	// TODO: use IP.IsLoopback() || IP.IsUnspecified()
	isForwardTunnel := forwardingDetails.Addr != "127.0.0.1" && forwardingDetails.Addr != "0.0.0.0" && forwardingDetails.Addr != "localhost"

	if isForwardTunnel {
		/* from RFC:
		"When a connection comes to a locally forwarded TCP/IP port, the
		following packet is sent to the other side.  Note that these messages
		MAY also be sent for ports for which no forwarding has been
		explicitly requested." */

		// we haven't implemented this part of the spec yet. PuTTY does not do this.
		log.Debug("client requesting pre-emptive forward even though it's not required")
		req.Reply(true, nil)
		return
	}

	cancelCh := fwdList.add(forwardingDetails)
	if cancelCh == nil {
		log.Error("TCP/IP reverse forward already reserved")
		req.Reply(false, nil)
		return
	}

	go processOnePortReverseRequest(
		forwardingDetails,
		req,
		serverConn,
		fwdList,
		*cancelCh)
}

func processTcpipCancelForwardReq(req *ssh.Request, fwdList *forwardList) {
	var cancelForwardDetails channelForwardMsg
	if err := ssh.Unmarshal(req.Payload, &cancelForwardDetails); err != nil {
		log.Error(err.Error())
		req.Reply(false, nil)
		return
	}

	if fwdList.cancel(cancelForwardDetails) {
		req.Reply(true, nil)
	} else {
		log.Error("cancel request for non-existent port")
		req.Reply(false, nil)
	}
}

// does same for ssh.NewChannel as above ProcessPortForwardRequests() does for ssh.Request
func ProcessPortForwardNewChannelRequests(newChannelRequests <-chan ssh.NewChannel) <-chan ssh.NewChannel {
	nonForwardNewChannels := make(chan ssh.NewChannel, 1)

	go func() {
		for newChannel := range newChannelRequests {
			switch newChannel.ChannelType() {
			case "direct-tcpip":
				var forwardingDetails channelOpenDirectMsg
				if err := ssh.Unmarshal(newChannel.ExtraData(), &forwardingDetails); err != nil {
					log.Error(err.Error())
					newChannel.Reject(ssh.UnknownChannelType, "payload unmarshal failed")
					continue
				}

				go processOnePortForwardRequest(forwardingDetails, newChannel)
			default:
				nonForwardNewChannels <- newChannel
			}
		}
	}()

	return nonForwardNewChannels
}

func processOnePortReverseRequest(
	forwardingDetails channelForwardMsg,
	req *ssh.Request,
	serverConn *ssh.ServerConn,
	fwdList *forwardList,
	cancel <-chan bool,
) {
	listenAddr := fmt.Sprintf("%s:%d", forwardingDetails.Addr, forwardingDetails.Rport)

	log.Info(fmt.Sprintf("Adding reverse listener to %s", listenAddr))

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Error(err.Error())
		req.Reply(false, nil)
		return
	}
	defer log.Info(fmt.Sprintf("Removed reverse listener %s", listenAddr))
	defer listener.Close()

	go func() {
		for {
			connToForward, err := listener.Accept()
			if err != nil {
				log.Error(fmt.Sprintf("Accept() failed: %s", err.Error()))
				fwdList.cancel(forwardingDetails)
				return
			}

			log.Debug(fmt.Sprintf("new client: %s", connToForward.RemoteAddr().String()))

			go func() {
				if err := forwardOneReverseConnection(serverConn, connToForward, forwardingDetails); err != nil {
					log.Error(fmt.Sprintf("processOnePortReverseRequest(): %s", err.Error()))
				}
			}()
		}
	}()

	go func() {
		// returns when SSH connection exists
		serverConn.Wait()

		fwdList.cancel(forwardingDetails)
	}()

	/*	FIXME: we probably should implement to-spec where responding with port if port in req was 0

		type channelForwardResponse struct {
			Port uint32
		}
	*/
	req.Reply(true, nil)

	// wait until reverse forward is: (all signalled via fwdList.cancel())
	// - cancelled explicitly by the client or
	// - the connection breaks
	// - listener.Accept() fails

	<-cancel
}

func forwardOneReverseConnection(sshServerConn *ssh.ServerConn, connToForward net.Conn, forwardingDetails channelForwardMsg) error {
	remoteAddr := connToForward.RemoteAddr()
	remoteHost, remotePortStr, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		return err
	}

	remotePort, err := strconv.Atoi(remotePortStr)
	if err != nil {
		return err
	}

	fordwardedMsg := &forwardedTCPPayload{
		Addr:       forwardingDetails.Addr,
		Port:       forwardingDetails.Rport,
		OriginAddr: remoteHost,
		OriginPort: uint32(remotePort),
	}

	// TCP stream is modeled as a SSH channel. it conveniently implements
	// io.ReadWriteCloser so we can just pipe the TCP connection and SSH channel in both directions
	tcpStreamCh, reqs, err := sshServerConn.OpenChannel("forwarded-tcpip", ssh.Marshal(fordwardedMsg))
	if err != nil {
		return err
	}

	// we're not expecting any requests for this channel
	go ssh.DiscardRequests(reqs)

	return bidipipe.Pipe(tcpStreamCh, "SSH tunnel", connToForward, "Local connection")
}

func processOnePortForwardRequest(forwardingDetails channelOpenDirectMsg, newChannel ssh.NewChannel) {
	remoteAddr := fmt.Sprintf("%s:%d", forwardingDetails.Raddr, forwardingDetails.Rport)

	log.Info(fmt.Sprintf("forwarding %s", remoteAddr))
	defer log.Info("closing")

	rconn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		log.Error(err.Error())
		newChannel.Reject(ssh.ConnectionFailed, err.Error())
		return
	}
	defer rconn.Close()

	tcpStreamCh, reqs, err := newChannel.Accept()
	if err != nil {
		log.Error("channel Accept() failed")
		return
	}

	go ssh.DiscardRequests(reqs)

	if err := bidipipe.Pipe(tcpStreamCh, "SSH tunnel", rconn, "Local connection"); err != nil {
		log.Error(err.Error())
	}
}
