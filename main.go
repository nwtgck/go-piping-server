package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

type Receiver struct {
	responseWriterCh chan http.ResponseWriter
	sendFinishedCh   chan struct{}
}

type PipingServer struct {
	pathToReceiver map[string]*Receiver
}

func NewServer() *PipingServer {
	return &PipingServer{pathToReceiver: map[string]*Receiver{}}
}

func (s *PipingServer) Handler(responseWriter http.ResponseWriter, req *http.Request) {
	fmt.Println(req.Method)
	path := req.URL.Path

	// Set receiver if not found on the path
	if _, ok := s.pathToReceiver[path]; !ok {
		s.pathToReceiver[path] = &Receiver{
			responseWriterCh: make(chan http.ResponseWriter),
			sendFinishedCh:   make(chan struct{}),
		}
	}
	receiver := s.pathToReceiver[path]

	// TODO: should block collision (e.g. GET the same path twice)
	// TODO: should close if either sender or receiver closes
	switch req.Method {
	case "GET":
		go func() { receiver.responseWriterCh <- responseWriter }()
		// Wait for finish
		<-receiver.sendFinishedCh
	case "POST":
	case "PUT":
		receiverResWriter := <-receiver.responseWriterCh
		// TODO: Hard code: content-type
		receiverResWriter.Header().Add("Content-Type", "application/octet-stream")
		io.Copy(receiverResWriter, req.Body)
		receiver.sendFinishedCh <- struct{}{}
		delete(s.pathToReceiver, path)
	}
	fmt.Printf("Trasfering %s has finished in %s method.\n", req.URL.Path, req.Method)
}

func main() {
	pipingServer := NewServer()
	fmt.Println("Running...")
	// TODO: Hard code port number
	log.Fatal(http.ListenAndServe(":8080", http.HandlerFunc(pipingServer.Handler)))
}
