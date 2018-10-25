FROM alpine:3.8

CMD ["holepunch-server", "server", "--http-reverse-proxy", "--sshd-websocket"]

ADD rel/holepunch-server_linux-amd64 /usr/local/bin/holepunch-server
