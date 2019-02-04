package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

type Receiver struct {
	receiverChan chan http.ResponseWriter
	finishedChan chan bool
}

var pathToReceiver map[string]*Receiver


func init() {
	// Initialize map
	pathToReceiver = map[string]*Receiver{}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method)
	path := r.URL.Path

	if _, ok  := pathToReceiver[path]; !ok {
		pathToReceiver[path] = &Receiver{
			receiverChan: make(chan http.ResponseWriter),
			finishedChan: make(chan bool),
		}
	}

	sr := pathToReceiver[path]

	// TODO: should block collision (e.g. GET the same path twice)
	// TODO: should close if either sender or receiver closes
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
		delete(pathToReceiver, path)
	}
	fmt.Printf("Trasfering %s has finished in %s method.\n", r.URL.Path, r.Method)
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("Running...")
	// TODO: Hard code port number
	log.Fatal(http.ListenAndServe(":8080", nil))
}
