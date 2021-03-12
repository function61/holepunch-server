FROM alpine:latest

# add SSH client
RUN apk add --update openssh

CMD ["holepunch-server", "server", "--http-reverse-proxy", "--sshd-websocket"]

ADD rel/holepunch-server_linux-amd64 /usr/local/bin/holepunch-server
