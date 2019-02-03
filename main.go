package main

import (
    "fmt"
	"io"
	"log"
    "net/http"
)

type SenderReceiver struct {
	senderChan   chan *http.Request
	receiverChan chan http.ResponseWriter
}

var pathToSenderReceiver map[string]*SenderReceiver


func init() {
	// Initialize map
	pathToSenderReceiver = map[string]*SenderReceiver{}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method)
	path := r.URL.Path

	if pathToSenderReceiver[path] == nil {
		pathToSenderReceiver[path] = &SenderReceiver{
			senderChan: make(chan *http.Request),
			receiverChan: make(chan http.ResponseWriter),
		}
	}

	sr := pathToSenderReceiver[path]

	// TODO: should check collision (e.g. GET the same path twice)
	switch r.Method {
	case "GET":
		go func(){ sr.receiverChan <- w }()
		sender := <-sr.senderChan
		// TODO: Hard code: content-type
		w.Header().Add("Content-Type", "application/octet-stream")
		io.Copy(w, sender.Body)
	case "POST":
	case "PUT":
		go func (){ sr.senderChan <- r }()
		receiver := <-sr.receiverChan
		// TODO: Hard code: content-type
		receiver.Header().Add("Content-Type", "application/octet-stream")
		io.Copy(receiver, r.Body)
	}
	fmt.Printf("Trasfering %s has finished in %s method.\n", r.URL.Path, r.Method)
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("Running...")
	// TODO: Hard code port number
	log.Fatal(http.ListenAndServe(":8080", nil))
}
