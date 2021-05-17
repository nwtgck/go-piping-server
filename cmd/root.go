package cmd

import (
	"errors"
	"fmt"
	"github.com/nwtgck/go-piping-server"
	"github.com/nwtgck/go-piping-server/version"
	"github.com/spf13/cobra"
	"net/http"
	"os"
)

var showsVersion bool
var httpPort uint16
var enableHttps bool
var httpsPort uint16
var keyPath string
var crtPath string

func init() {
	cobra.OnInitialize()
	RootCmd.PersistentFlags().BoolVarP(&showsVersion, "version", "", false, "show version")
	RootCmd.PersistentFlags().Uint16VarP(&httpPort, "http-port", "", 8080, "HTTP port")
	RootCmd.PersistentFlags().BoolVarP(&enableHttps, "enable-https", "", false, "Enable HTTPS")
	RootCmd.PersistentFlags().Uint16VarP(&httpsPort, "https-port", "", 8443, "HTTPS port")
	RootCmd.PersistentFlags().StringVarP(&keyPath, "key-path", "", "", "Private key path")
	RootCmd.PersistentFlags().StringVarP(&crtPath, "crt-path", "", "", "Certification path")
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
		pipingServer := piping_server.NewServer()
		errCh := make(chan error)
		if enableHttps {
			if keyPath == "" {
				return errors.New("key-path should be specified")
			}
			if crtPath == "" {
				return errors.New("crt-path should be specified")
			}
			go func() {
				fmt.Printf("Listening HTTPS on %d...\n", httpsPort)
				errCh <- http.ListenAndServeTLS(fmt.Sprintf(":%d", httpsPort), crtPath, keyPath, http.HandlerFunc(pipingServer.Handler))
			}()
		}
		go func() {
			fmt.Printf("Listening HTTP on %d...\n", httpPort)
			errCh <- http.ListenAndServe(fmt.Sprintf(":%d", httpPort), http.HandlerFunc(pipingServer.Handler))
		}()
		return <-errCh
	},
}
