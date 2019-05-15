package reverseproxy

import (
	"errors"
	"fmt"
	"github.com/function61/gokit/logex"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strconv"
)

var disallowedPorts = []int{22, 80, 443, 8080}

func Register(mux *http.ServeMux, logger *log.Logger) {
	logl := logex.Levels(logger)

	reverseProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			destinationPort, err := destinationPortFromVirtualHost(req.Host)
			if err != nil {
				logl.Error.Println(err.Error())

				// leaving Scheme unset aborts the request gracefully
			} else {
				req.URL.Scheme = "http"
				req.URL.Host = fmt.Sprintf("localhost:%d", destinationPort)
			}
		},
	}

	mux.Handle("/", reverseProxy)
}

// 8081.punch.fn61.net => 8081
var destinationPortRe = regexp.MustCompile("^([0-9]+)\\.")

func destinationPortFromVirtualHost(virtualHost string) (int, error) {
	matches := destinationPortRe.FindStringSubmatch(virtualHost)
	if matches == nil {
		return 0, errors.New("failed to determine destination port from vhost")
	}

	destinationPort, err := strconv.Atoi(matches[1])
	if err != nil { // should not happen
		return 0, err
	}

	if isDisallowedPort(destinationPort) {
		return 0, errors.New("destination port is disallowed")
	}

	return destinationPort, nil
}

func isDisallowedPort(port int) bool {
	for _, disallowedPort := range disallowedPorts {
		if port == disallowedPort {
			return true
		}
	}

	return false
}
