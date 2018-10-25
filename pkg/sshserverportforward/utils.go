package sshserverportforward

import (
	"fmt"
	"golang.org/x/crypto/ssh"
)

// since you're embedding port forwarding in a SSH server, there's a good chance you might
// need to discard channel requests from client -> server, since they're mainly used for
// interactive sessions
func RejectChannelRequests(channelRequests <-chan ssh.NewChannel) {
	for channelRequest := range channelRequests {
		channelRequest.Reject(ssh.Prohibited, fmt.Sprintf("channel type prohibited: %s", channelRequest.ChannelType()))
	}
}
