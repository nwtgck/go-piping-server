package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

type SenderReceiver struct {
	receiverChan chan http.ResponseWriter
	finishedChan chan bool
}

var pathToSenderReceiver map[string]*SenderReceiver


func init() {
	// Initialize map
	pathToSenderReceiver = map[string]*SenderReceiver{}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method)
	path := r.URL.Path

	if _, ok  := pathToSenderReceiver[path]; !ok {
		pathToSenderReceiver[path] = &SenderReceiver{
			receiverChan: make(chan http.ResponseWriter),
			finishedChan: make(chan bool),
		}
	}

	sr := pathToSenderReceiver[path]

	// TODO: should check collision (e.g. GET the same path twice)
	switch r.Method {
	case "GET":
		go func(){ sr.receiverChan <- w }()
		// Wait for finish
		<-sr.finishedChan
	case "POST":
	case "PUT":
		receiver := <-sr.receiverChan
		// TODO: Hard code: content-type
		receiver.Header().Add("Content-Type", "application/octet-stream")
		io.Copy(receiver, r.Body)
		sr.finishedChan <- true
		delete(pathToSenderReceiver, path)
	}
	fmt.Printf("Trasfering %s has finished in %s method.\n", r.URL.Path, r.Method)
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("Running...")
	// TODO: Hard code port number
	log.Fatal(http.ListenAndServe(":8080", nil))
}
