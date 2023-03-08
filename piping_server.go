package piping_server

import (
	"fmt"
	"github.com/nwtgck/go-piping-server/version"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"sync"
	"sync/atomic"
)

const (
	reservedPathIndex      = "/"
	reservedPathNoScript   = "/noscript"
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

const noscriptPathQueryParameterName = "path"

type pipe struct {
	receiverResWriterCh chan http.ResponseWriter
	sendFinishedCh      chan struct{}
	isSenderConnected   uint32 // NOTE: for atomic operation
	isTransferring      uint32 // NOTE: for atomic operation
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

func transferHeaderIfExists(w http.ResponseWriter, reqHeader textproto.MIMEHeader, header string) {
	values := reqHeader.Values(header)
	if len(values) == 1 {
		w.Header().Add(header, values[0])
	}
}

func getTransferHeaderAndBody(req *http.Request) (textproto.MIMEHeader, io.ReadCloser) {
	mediaType, params, mediaTypeParseErr := mime.ParseMediaType(req.Header.Get("Content-Type"))
	// If multipart upload
	if mediaTypeParseErr == nil && mediaType == "multipart/form-data" {
		multipartReader := multipart.NewReader(req.Body, params["boundary"])
		part, err := multipartReader.NextPart()
		if err != nil {
			// Return normal if multipart error
			return textproto.MIMEHeader(req.Header), req.Body
		}
		return part.Header, part
	}
	return textproto.MIMEHeader(req.Header), req.Body
}

func (s *PipingServer) Handler(resWriter http.ResponseWriter, req *http.Request) {
	s.logger.Printf("%s %s %s", req.Method, req.URL, req.Proto)
	path := req.URL.Path

	if req.Method == "GET" || req.Method == "HEAD" {
		switch path {
		case reservedPathIndex:
			indexPageBytes := []byte(indexPage)
			resWriter.Header().Set("Content-Type", "text/html")
			resWriter.Header().Set("Content-Length", strconv.Itoa(len(indexPageBytes)))
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.Write(indexPageBytes)
			return
		case reservedPathNoScript:
			noScriptHtmlBytes := []byte(noScriptHtml(req.URL.Query().Get(noscriptPathQueryParameterName)))
			resWriter.Header().Set("Content-Type", "text/html")
			resWriter.Header().Set("Content-Length", strconv.Itoa(len(noScriptHtmlBytes)))
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.Write(noScriptHtmlBytes)
			return
		case reservedPathVersion:
			versionBytes := []byte(fmt.Sprintf("%s in Go\n", version.Version))
			resWriter.Header().Set("Content-Type", "text/plain")
			resWriter.Header().Set("Content-Length", strconv.Itoa(len(versionBytes)))
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.Write(versionBytes)
			return
		case reservedPathHelp:
			protocol := "http"
			if req.TLS != nil {
				protocol = "https"
			}
			url := fmt.Sprintf(protocol+"://%s", req.Host)
			helpPageBytes := []byte(helpPage(url))
			resWriter.Header().Set("Content-Type", "text/plain")
			resWriter.Header().Set("Content-Length", strconv.Itoa(len(helpPageBytes)))
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.Write(helpPageBytes)
			return
		case reservedPathFaviconIco:
			resWriter.Header().Set("Content-Length", "0")
			resWriter.WriteHeader(204)
			return
		case reservedPathRobotsTxt:
			resWriter.Header().Set("Content-Length", "0")
			resWriter.WriteHeader(404)
			return
		}
	}

	// TODO: should close if either sender or receiver closes
	switch req.Method {
	case "GET":
		// If the receiver requests Service Worker registration
		// (from: https://speakerdeck.com/masatokinugawa/pwa-study-sw?slide=32)
		if req.Header.Get("Service-Worker") == "script" {
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.WriteHeader(400)
			resWriter.Write([]byte("[ERROR] Service Worker registration is rejected.\n"))
			return
		}
		pi := s.getPipe(path)
		// If already get the path or transferring
		if len(pi.receiverResWriterCh) != 0 || atomic.LoadUint32(&pi.isTransferring) == 1 {
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
		// Notify that Content-Range is not supported
		// In the future, resumable upload using Content-Range might be supported
		// ref: https://github.com/httpwg/http-core/pull/653
		if len(req.Header.Values("Content-Range")) != 0 {
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.WriteHeader(400)
			resWriter.Write([]byte(fmt.Sprintf("[ERROR] Content-Range is not supported for now in %s\n", req.Method)))
			return
		}
		pi := s.getPipe(path)
		// If a sender is already connected
		if !atomic.CompareAndSwapUint32(&pi.isSenderConnected, 0, 1) {
			resWriter.Header().Set("Access-Control-Allow-Origin", "*")
			resWriter.WriteHeader(400)
			resWriter.Write([]byte(fmt.Sprintf("[ERROR] Another sender has been connected on '%s'.\n", path)))
			return
		}

		contentLength := req.ContentLength
		// NOTE: `req.ContentLength = 0` is a workaround for full duplex
		// Replace with https://github.com/golang/go/blob/457fd1d52d17fc8e73d4890150eadab3128de64d/src/net/http/responsecontroller.go#L119-L141 in the future
		req.ContentLength = 0
		resWriter.Header().Set("Access-Control-Allow-Origin", "*")
		resWriter.WriteHeader(200)
		if f, ok := resWriter.(http.Flusher); ok {
			f.Flush()
		}
		req.ContentLength = contentLength

		resWriteFlusher := NewWriteFlusherIfPossible(resWriter)
		if _, err := resWriteFlusher.Write([]byte("[INFO] Waiting for 1 receiver(s)...\n")); err != nil {
			return
		}
		receiverResWriter := <-pi.receiverResWriterCh
		if _, err := resWriteFlusher.Write([]byte("[INFO] A receiver was connected.\n")); err != nil {
			return
		}
		if _, err := resWriteFlusher.Write([]byte("[INFO] Start sending to 1 receiver(s)!\n")); err != nil {
			return
		}
		atomic.StoreUint32(&pi.isTransferring, 1)
		transferHeader, transferBody := getTransferHeaderAndBody(req)
		receiverResWriter.Header()["Content-Type"] = nil // not to sniff
		transferHeaderIfExists(receiverResWriter, transferHeader, "Content-Type")
		transferHeaderIfExists(receiverResWriter, transferHeader, "Content-Length")
		transferHeaderIfExists(receiverResWriter, transferHeader, "Content-Disposition")
		xPipingValues := req.Header.Values("X-Piping")
		if len(xPipingValues) != 0 {
			receiverResWriter.Header()["X-Piping"] = xPipingValues
		}
		receiverResWriter.Header().Set("Access-Control-Allow-Origin", "*")
		if len(xPipingValues) != 0 {
			receiverResWriter.Header().Set("Access-Control-Expose-Headers", "X-Piping")
		}
		receiverResWriter.Header().Set("X-Robots-Tag", "none")
		receiverResWriteFlusher := NewWriteFlusherIfPossible(receiverResWriter)
		if _, err := io.Copy(receiverResWriteFlusher, transferBody); err != nil {
			return
		}
		if _, err := resWriteFlusher.Write([]byte("[INFO] Sent successfully!\n")); err != nil {
			return
		}
		pi.sendFinishedCh <- struct{}{}
		delete(s.pathToPipe, path)
	case "OPTIONS":
		resWriter.Header().Set("Access-Control-Allow-Origin", "*")
		resWriter.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, OPTIONS")
		resWriter.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Disposition, X-Piping")
		resWriter.Header().Set("Access-Control-Max-Age", "86400")
		resWriter.Header().Set("Content-Length", "0")
		resWriter.WriteHeader(200)
		return
	default:
		resWriter.WriteHeader(405)
		resWriter.Header().Set("Access-Control-Allow-Origin", "*")
		resWriter.Write([]byte(fmt.Sprintf("[ERROR] Unsupported method: %s.\n", req.Method)))
		return
	}
	s.logger.Printf("Transferring %s has finished in %s method.\n", req.URL.Path, req.Method)
}

type WriteFlusher struct {
	writer  io.Writer
	flusher http.Flusher
}

func NewWriteFlusherIfPossible(w http.ResponseWriter) io.Writer {
	if f, ok := w.(http.Flusher); ok {
		return &WriteFlusher{writer: w, flusher: f}
	}
	return w
}

func (f *WriteFlusher) Write(p []byte) (int, error) {
	n, err := f.writer.Write(p)
	if err != nil {
		return n, err
	}
	f.flusher.Flush()
	return n, err
}
