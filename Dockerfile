FROM golang:1.16
COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 go build -o piping-server main/main.go

FROM scratch
LABEL maintainer="Ryo Ota <nwtgck@nwtgck.org>"
COPY --from=0 /app/piping-server /usr/local/bin/piping-server
ENTRYPOINT ["/usr/local/bin/piping-server"]
