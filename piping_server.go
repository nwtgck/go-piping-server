package piping_server

import (
	"fmt"
	"github.com/nwtgck/go-piping-server/version"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
)

const (
	reservedPathIndex      = "/"
	reservedPathVersion    = "/version"
	reservedPathHelp       = "/help"
	reservedPathFaviconIco = "/favicon.ico"
	reservedPathRobotsTxt  = "/robots.txt"
)

var reservedPaths = [...]string{
	reservedPathIndex,
	reservedPathVersion,
	reservedPathHelp,
	reservedPathFaviconIco,
	reservedPathRobotsTxt,
}

type pipe struct {
	receiverResWriterCh chan http.ResponseWriter
	sendFinishedCh      chan struct{}
	isSenderConnected   int32 // NOTE: for atomic operation
}

type PipingServer struct {
	pathToPipe map[string]*pipe
	mutex      *sync.Mutex
	logger     *log.Logger
}

func isReservedPath(path string) bool {
	for _, p := range reservedPaths {
		if p == path {
			return true
		}
	}
	return false
}

func NewServer(logger *log.Logger) *PipingServer {
	return &PipingServer{
		pathToPipe: map[string]*pipe{},
		mutex:      new(sync.Mutex),
		logger:     logger,
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

func transferHeaderIfExists(w http.ResponseWriter, req *http.Request, header string) {
	values := req.Header.Values(header)
	if len(values) == 1 {
		w.Header().Add(header, values[0])
	}
}

func (s *PipingServer) Handler(resWriter http.ResponseWriter, req *http.Request) {
	s.logger.Printf("%s %s", req.Method, req.URL)
	path := req.URL.Path

	// TODO: should close if either sender or receiver closes
	switch req.Method {
	case "GET":
		switch path {
		case reservedPathIndex:
			resWriter.Header().Set("Content-Type", "text/html")
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.Write([]byte(indexPage))
			return
		case reservedPathVersion:
			resWriter.Header().Set("Content-Type", "text/plain")
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.Write([]byte(fmt.Sprintf("%s in Go\n", version.Version)))
			return
		case reservedPathHelp:
			resWriter.Header().Set("Content-Type", "text/plain")
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			protocol := "http"
			if req.TLS != nil {
				protocol = "https"
			}
			url := fmt.Sprintf(protocol+"://%s", req.Host)
			resWriter.Write([]byte(helpPage(url)))
			return
		case reservedPathFaviconIco:
			resWriter.WriteHeader(204)
			return
		case reservedPathRobotsTxt:
			resWriter.WriteHeader(404)
			return
		}
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
	case "POST", "PUT":
		// If reserved path
		if isReservedPath(path) {
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.WriteHeader(400)
			resWriter.Write([]byte(fmt.Sprintf("[ERROR] Cannot send to the reserved path '%s'. (e.g. '/mypath123')\n", path)))
			return
		}
		pi := s.getPipe(path)
		// If a sender is already connected
		if !atomic.CompareAndSwapInt32(&pi.isSenderConnected, 0, 1) {
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.WriteHeader(400)
			resWriter.Write([]byte(fmt.Sprintf("[ERROR] Another sender has been connected on '%s'.\n", path)))
			return
		}
		receiverResWriter := <-pi.receiverResWriterCh
		receiverResWriter.Header()["Content-Type"] = nil // not to sniff
		transferHeaderIfExists(receiverResWriter, req, "Content-Type")
		transferHeaderIfExists(receiverResWriter, req, "Content-Length")
		transferHeaderIfExists(receiverResWriter, req, "Content-Disposition")
		receiverResWriter.Header().Set("Access-Control-Allow-Origin", "*")
		receiverResWriter.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		io.Copy(receiverResWriter, req.Body)
		pi.sendFinishedCh <- struct{}{}
		delete(s.pathToPipe, path)
	case "OPTIONS":
		resWriter.Header().Set("Access-Control-Allow-Origin", "*")
		resWriter.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, OPTIONS")
		resWriter.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Disposition")
		resWriter.Header().Set("Access-Control-Max-Age", "86400")
		resWriter.Header().Set("Content-Length", "0")
		resWriter.WriteHeader(200)
		return
	default:
		resWriter.WriteHeader(400)
		resWriter.Write([]byte(fmt.Sprintf("[ERROR] Unsupported method: %s.\n", req.Method)))
	}
	s.logger.Printf("Transferring %s has finished in %s method.\n", req.URL.Path, req.Method)
}
