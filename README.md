[![Build Status](https://img.shields.io/travis/function61/holepunch-server.svg?style=for-the-badge)](https://travis-ci.org/function61/holepunch-server)
[![Download](https://img.shields.io/bintray/v/function61/holepunch-server/main.svg?style=for-the-badge&label=Download)](https://bintray.com/function61/holepunch-server/main/_latestVersion#files)
[![Download](https://img.shields.io/docker/pulls/fn61/holepunch-server.svg?style=for-the-badge)](https://hub.docker.com/r/fn61/holepunch-server/)

holepunch-server
================

Don't you just hate when you want to connect an IoT device behind a mobile connection
to the internet, and you notice you are behind a
[carrier-grade NAT](https://en.wikipedia.org/wiki/Carrier-grade_NAT), and the ISP tries to
persuade you to upgrade to a more expensive plan just to have a public IP?

You can use a SSH reverse tunnel to expose the device's port to internet if you have a
server somewhere. This program takes it a bit further and allows you to multiplex many
different IP:s/ports behind a single HTTP server port, so you don't need to open any ports
in your firewall. Of course this approach works only for those TCP ports that speak HTTP.
And you can use any standard loadbalancer in front of this, if you have edge routing for example.

Essentially this program uses virtual hosting to decide which TCP port to forward traffic to.

![Architecture](docs/architecture.png)


Usage
-----

Download suitable binary for your architecture from the download link.

Generate server host key, then output it as base64:

```
$ ssh-keygen -t ecdsa -b 521 -C "my awesome private key" -f id_ecdsa
$ cat id_ecdsa | base64 -w 0
LS0tLS1CRUdJTi...
```

This will be your ENV variable `SSH_HOSTKEY`

The other ENV variable will be `CLIENT_PUBKEY`. This won't need to be base64 encoded.

The content of that variable you can find from file `id_ecdsa.pub` for the client
([example](https://github.com/function61/holepunch-client#usage)).

Now set up ENV vars and start `holepunch-server`:

```
export SSH_HOSTKEY="..."
export CLIENT_PUBKEY="..."
$ ./holepunch-server server --sshd-websocket --http-reverse-proxy --sshd-tcp 0.0.0.0:22
```

The above command line is if you want all the bells and whistles. If your clients will be
using only Websocket, you might want to disable the TCP port for reduced attack surface.

This is also available as a Docker image, which by default only enables SSH Websocket and
HTTP reverse proxy. You need to configure the ENV vars via your favourite deployment tool.
