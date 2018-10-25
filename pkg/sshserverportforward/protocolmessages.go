package sshserverportforward

// these structs copy-pasted from golang.org/x/crypto/ssh/tcpip.go
// (couldn't link to them because they are private)

// RFC 4254 7.1
type channelForwardMsg struct {
	Addr  string
	Rport uint32
}

// See RFC 4254, section 7.2
type forwardedTCPPayload struct {
	Addr       string
	Port       uint32
	OriginAddr string
	OriginPort uint32
}
