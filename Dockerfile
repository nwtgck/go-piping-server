# NOTE: base platform is always linux/amd64 because go can cross-build
FROM --platform=linux/amd64 golang:1.16

ARG TARGETPLATFORM

COPY . /app
WORKDIR /app
RUN case $TARGETPLATFORM in\
      linux/amd64)  GOARCH="amd64";;\
      linux/arm/v6) GOARCH="arm"; GOARM=6;;\
      linux/arm/v7) GOARCH="arm"; GOARM=7;;\
      linux/arm64)  GOARCH="arm64";;\
      *)            exit 1;;\
    esac &&\
    GOOS=linux CGO_ENABLED=0 go build -o piping-server main/main.go

FROM scratch
LABEL maintainer="Ryo Ota <nwtgck@nwtgck.org>"
COPY --from=0 /app/piping-server /usr/local/bin/piping-server
ENTRYPOINT ["/usr/local/bin/piping-server"]
