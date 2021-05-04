package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

type pipe struct {
	receiverResWriterCh chan http.ResponseWriter
	sendFinishedCh      chan struct{}
}

type PipingServer struct {
	pathToPipe map[string]*pipe
}

func NewServer() *PipingServer {
	return &PipingServer{pathToPipe: map[string]*pipe{}}
}

func (s *PipingServer) Handler(resWriter http.ResponseWriter, req *http.Request) {
	fmt.Println(req.Method)
	path := req.URL.Path

	// Set pipe if not found on the path
	if _, ok := s.pathToPipe[path]; !ok {
		s.pathToPipe[path] = &pipe{
			receiverResWriterCh: make(chan http.ResponseWriter, 1),
			sendFinishedCh:      make(chan struct{}),
		}
	}
	pi := s.pathToPipe[path]

	// TODO: should block collision (e.g. POST the same path twice)
	// TODO: should close if either sender or receiver closes
	switch req.Method {
	case "GET":
		// If already get the path
		if len(pi.receiverResWriterCh) != 0 {
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.WriteHeader(400)
			resWriter.Write([]byte("[ERROR] The number of receivers has reached limits.\n"))
			return
		}
		go func() { pi.receiverResWriterCh <- resWriter }()
		// Wait for finish
		<-pi.sendFinishedCh
	case "POST":
		fallthrough
	case "PUT":
		receiverResWriter := <-pi.receiverResWriterCh
		// TODO: Hard code: content-type
		receiverResWriter.Header().Add("Content-Type", "application/octet-stream")
		io.Copy(receiverResWriter, req.Body)
		pi.sendFinishedCh <- struct{}{}
		delete(s.pathToPipe, path)
	}
	fmt.Printf("Trasfering %s has finished in %s method.\n", req.URL.Path, req.Method)
}

func main() {
	pipingServer := NewServer()
	fmt.Println("Running...")
	// TODO: Hard code port number
	log.Fatal(http.ListenAndServe(":8080", http.HandlerFunc(pipingServer.Handler)))
}
