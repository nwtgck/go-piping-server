# go-piping-server
[Piping Server](https://github.com/nwtgck/piping-server) written in Go language (original: <https://github.com/nwtgck/piping-server>)

## Install for Ubuntu
```bash
wget https://github.com/nwtgck/go-piping-server/releases/download/v0.6.2/go-piping-server-0.6.2-linux-amd64.deb
sudo dpkg -i go-piping-server-0.6.2-linux-amd64.deb
```

## Install for macOS
```bash
brew install nwtgck/go-piping-server/go-piping-server
```

## Install for Windows
[Download](https://github.com/nwtgck/go-piping-server/releases/download/v0.6.2/go-piping-server-0.6.2-windows-amd64.zip)

Get more executables in the [releases](https://github.com/nwtgck/go-piping-server/releases).

## Docker

```bash
docker run -p 8181:8080 nwtgck/go-piping-server
```

## Server options

```
Infinitely transfer between any device over pure HTTP

Usage:
  go-piping-server [flags]

Flags:
      --crt-path string     Certification path
      --enable-http3        Enable HTTP/3 (experimental)
      --enable-https        Enable HTTPS
  -h, --help                help for go-piping-server
      --http-port uint16    HTTP port (default 8080)
      --https-port uint16   HTTPS port (default 8443)
      --key-path string     Private key path
      --version             show version
```
