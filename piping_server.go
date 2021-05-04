package piping_server

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
)

type pipe struct {
	receiverResWriterCh chan http.ResponseWriter
	sendFinishedCh      chan struct{}
	isSenderConnected   int32 // NOTE: for atomic operation
}

type PipingServer struct {
	pathToPipe map[string]*pipe
	mutex      *sync.Mutex
}

func NewServer() *PipingServer {
	return &PipingServer{
		pathToPipe: map[string]*pipe{},
		mutex:      new(sync.Mutex),
	}
}

func (s *PipingServer) getPipe(path string) *pipe {
	// Set pipe if not found on the path
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.pathToPipe[path]; !ok {
		pi := &pipe{
			receiverResWriterCh: make(chan http.ResponseWriter, 1),
			sendFinishedCh:      make(chan struct{}),
			isSenderConnected:   0,
		}
		s.pathToPipe[path] = pi
		return pi
	}
	return s.pathToPipe[path]
}

func (s *PipingServer) Handler(resWriter http.ResponseWriter, req *http.Request) {
	fmt.Println(req.Method)
	path := req.URL.Path

	// TODO: should close if either sender or receiver closes
	switch req.Method {
	case "GET":
		pi := s.getPipe(path)
		// If already get the path
		if len(pi.receiverResWriterCh) != 0 {
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.WriteHeader(400)
			resWriter.Write([]byte("[ERROR] The number of receivers has reached limits.\n"))
			return
		}
		pi.receiverResWriterCh <- resWriter
		// Wait for finish
		<-pi.sendFinishedCh
	case "POST":
		fallthrough
	case "PUT":
		pi := s.getPipe(path)
		// If a sender is already connected
		if !atomic.CompareAndSwapInt32(&pi.isSenderConnected, 0, 1) {
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.WriteHeader(400)
			resWriter.Write([]byte(fmt.Sprintf("[ERROR] Another sender has been connected on '%s'.\n", path)))
			return
		}
		receiverResWriter := <-pi.receiverResWriterCh
		// TODO: Hard code: content-type
		receiverResWriter.Header().Add("Content-Type", "application/octet-stream")
		io.Copy(receiverResWriter, req.Body)
		pi.sendFinishedCh <- struct{}{}
		delete(s.pathToPipe, path)
	}
	fmt.Printf("Trasfering %s has finished in %s method.\n", req.URL.Path, req.Method)
}
