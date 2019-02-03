package main

import (
    "fmt"
	"io"
	"log"
    "net/http"
)

// TODO: This adapt only one pipe
var receiverChan chan http.ResponseWriter
var senderChan chan *http.Request

func init() {
	receiverChan = make(chan http.ResponseWriter)
	senderChan = make(chan *http.Request)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method)
	switch r.Method {
	case "GET":
		go func(){ receiverChan <- w }()
		sender := <-senderChan
		// TODO: Hard code: content-type
		w.Header().Add("Content-Type", "application/octet-stream")
		io.Copy(w, sender.Body)
	case "POST":
	case "PUT":
		go func (){ senderChan <- r }()
		receiver := <-receiverChan
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
