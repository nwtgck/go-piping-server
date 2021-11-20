package main

import (
	"fmt"
	"github.com/nwtgck/go-piping-server/cmd"
	"os"
)

func main() {
	//if err := cmd.RootCmd.Execute(); err != nil {
	//	_, _ = fmt.Fprintf(os.Stderr, err.Error())
	//	os.Exit(-1)
	//}

	if err := cmd.App.Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(-1)
	}
}
