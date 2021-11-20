package cmd

import (
	"errors"
	"fmt"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/nwtgck/go-piping-server"
	"github.com/nwtgck/go-piping-server/version"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"log"
	"net/http"
	"os"
)

var showsVersion bool
var httpPort uint
var enableHttps bool
var httpsPort uint
var keyPath string
var crtPath string
var enableHttp3 bool

var App = &cli.App{
	Usage: "Infinitely transfer between any device over pure HTTP",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "version", Usage: "Show version", Destination: &showsVersion},
		&cli.UintFlag{Name: "http-port", Usage: "HTTP port", Value: 8080, Destination: &httpPort},
		&cli.BoolFlag{Name: "enable-https", Usage: "Enable HTTPS", Value: false, Destination: &enableHttps},
		&cli.UintFlag{Name: "https-port", Usage: "HTTPS port", Value: 8443, Destination: &httpsPort},
		&cli.StringFlag{Name: "key-path", Usage: "Private key path", Destination: &keyPath},
		&cli.StringFlag{Name: "crt-path", Usage: "Certification path", Destination: &crtPath},
		&cli.BoolFlag{Name: "enable-http3", Usage: "Enable HTTP/3 (experimental)", Value: false, Destination: &enableHttp3},
	},
	// NOTE: No `Version: version.Version` because it adds `-v` short flag,
	HideHelpCommand: true,
	Action: func(context *cli.Context) error {
		if showsVersion {
			fmt.Println(version.Version)
			return nil
		}
		logger := log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
		pipingServer := piping_server.NewServer(logger)
		errCh := make(chan error)
		if enableHttps || enableHttp3 {
			if keyPath == "" {
				return errors.New("--key-path should be specified")
			}
			if crtPath == "" {
				return errors.New("--crt-path should be specified")
			}
			go func() {
				logger.Printf("Listening HTTPS on %d...\n", httpsPort)
				errCh <- http.ListenAndServeTLS(fmt.Sprintf(":%d", httpsPort), crtPath, keyPath, http.HandlerFunc(pipingServer.Handler))
			}()
			if enableHttp3 {
				go func() {
					logger.Printf("Listening HTTP/3 on %d...\n", httpsPort)
					errCh <- http3.ListenAndServeQUIC(fmt.Sprintf(":%d", httpsPort), crtPath, keyPath, http.HandlerFunc(pipingServer.Handler))
				}()
			}
		}
		go func() {
			server := &http.Server{
				Addr:    fmt.Sprintf(":%d", httpPort),
				Handler: h2c.NewHandler(http.HandlerFunc(pipingServer.Handler), &http2.Server{}),
			}
			logger.Printf("Listening HTTP on %d...\n", httpPort)
			errCh <- server.ListenAndServe()
		}()
		return <-errCh
	},
}
