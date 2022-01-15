package cmd

import (
	"errors"
	"fmt"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/nwtgck/go-piping-server"
	"github.com/nwtgck/go-piping-server/version"
	"github.com/spf13/cobra"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"log"
	"net/http"
	"os"
)

var showsVersion bool
var httpPort uint16
var enableHttps bool
var httpsPort uint16
var keyPath string
var crtPath string
var enableHttp3 bool

func init() {
	cobra.OnInitialize()
	RootCmd.PersistentFlags().BoolVarP(&showsVersion, "version", "", false, "show version")
	RootCmd.PersistentFlags().Uint16VarP(&httpPort, "http-port", "", 8080, "HTTP port")
	RootCmd.PersistentFlags().BoolVarP(&enableHttps, "enable-https", "", false, "Enable HTTPS")
	RootCmd.PersistentFlags().Uint16VarP(&httpsPort, "https-port", "", 8443, "HTTPS port")
	RootCmd.PersistentFlags().StringVarP(&keyPath, "key-path", "", "", "Private key path")
	RootCmd.PersistentFlags().StringVarP(&crtPath, "crt-path", "", "", "Certification path")
	RootCmd.PersistentFlags().BoolVarP(&enableHttp3, "enable-http3", "", false, "Enable HTTP/3 (experimental)")
}

var RootCmd = &cobra.Command{
	Use:          os.Args[0],
	Short:        "piping-server",
	Long:         "Infinitely transfer between any device over pure HTTP",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if showsVersion {
			fmt.Println(version.Version)
			return nil
		}
		logger := log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("Piping Server (Go) %s", version.Version)
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
