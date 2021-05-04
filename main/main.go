package main

import (
	"fmt"
	"github.com/nwtgck/go-piping-server"
	"log"
	"net/http"
)

func main() {
	pipingServer := piping_server.NewServer()
	fmt.Println("Running...")
	// TODO: Hard code port number
	log.Fatal(http.ListenAndServe(":8080", http.HandlerFunc(pipingServer.Handler)))
}
