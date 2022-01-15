package piping_server

import (
	"fmt"
	"github.com/nwtgck/go-piping-server/version"
	"golang.org/x/net/context"
	"gotest.tools/v3/assert"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// serve serves Piping Server on available port
func serve(t *testing.T) (*http.Server, string) {
	logger := log.New(io.Discard, "", log.LstdFlags|log.Lmicroseconds)
	pipingServer := NewServer(logger)
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(pipingServer.Handler)}
	go func() {
		err := server.Serve(ln)
		if err == http.ErrServerClosed {
			return
		}
		if err != nil {
			t.Error(err)
			return
		}
	}()
	return server, "http://" + ln.Addr().String()
}

func readerToString(t *testing.T, r io.Reader) string {
	stringBuilder := new(strings.Builder)
	_, err := io.Copy(stringBuilder, r)
	if err != nil {
		t.Fatal(err)
	}
	return stringBuilder.String()
}

func TestIndexPage(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	res, err := http.Get(url)
	if err != nil {
		t.Fatal(t)
	}
	assert.Equal(t, res.StatusCode, 200)
	assert.Equal(t, res.Header.Get("Content-Type"), "text/html")
	body := readerToString(t, res.Body)
	assert.Assert(t, strings.Contains(body, "Piping"))
}

func TestNoscriptPage(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	res, err := http.Get(url + "/noscript?path=mypath")
	if err != nil {
		t.Fatal(t)
	}
	assert.Equal(t, res.StatusCode, 200)
	assert.Equal(t, res.Header.Get("Content-Type"), "text/html")
	body := readerToString(t, res.Body)
	assert.Assert(t, strings.Contains(body, `action="mypath"`))
}

func TestVersionPage(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	res, err := http.Get(url + "/version")
	if err != nil {
		t.Fatal(t)
	}
	assert.Equal(t, res.StatusCode, 200)
	assert.Equal(t, res.Header.Get("Content-Type"), "text/plain")
	assert.Equal(t, res.Header.Get("Access-Control-Allow-Origin"), "*")
	body := readerToString(t, res.Body)
	assert.Equal(t, body, fmt.Sprintf("%s in Go\n", version.Version))
}

func TestHelpPage(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	res, err := http.Get(url + "/help")
	if err != nil {
		t.Fatal(t)
	}
	assert.Equal(t, res.StatusCode, 200)
	assert.Equal(t, res.Header.Get("Content-Type"), "text/plain")
	assert.Equal(t, res.Header.Get("Access-Control-Allow-Origin"), "*")
}

func TestFavicon(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	res, err := http.Get(url + "/favicon.ico")
	if err != nil {
		t.Fatal(t)
	}
	assert.Equal(t, res.StatusCode, 204)
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(t)
	}
	assert.Equal(t, len(bytes), 0)
}

func TestRobotsTxt(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	res, err := http.Get(url + "/robots.txt")
	if err != nil {
		t.Fatal(t)
	}
	assert.Equal(t, res.StatusCode, 404)
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(t)
	}
	assert.Equal(t, len(bytes), 0)
}

func TestHeadMethodInReservedPaths(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	for _, path := range reservedPaths {
		headRes, err := http.Head(url + path)
		if err != nil {
			t.Fatal(t)
		}
		getRes, err := http.Get(url + path)
		if err != nil {
			t.Fatal(t)
		}
		headRes.Header.Del("Date")
		getRes.Header.Del("Date")
		assert.Equal(t, headRes.StatusCode, getRes.StatusCode)
		assert.DeepEqual(t, headRes.Header, getRes.Header)
	}
}

func TestPreflightRequest(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	req, err := http.NewRequest("OPTIONS", url+"/mypath", nil)
	if err != nil {
		t.Fatal(t)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(t)
	}
	assert.Equal(t, res.StatusCode, 200)
	assert.Equal(t, res.Header.Get("Access-Control-Allow-Origin"), "*")
	assert.Equal(t, res.Header.Get("Access-Control-Allow-Methods"), "GET, HEAD, POST, PUT, OPTIONS")
	assert.Equal(t, strings.ToLower(res.Header.Get("Access-Control-Allow-Headers")), "content-type, content-disposition, x-piping")
	assert.Equal(t, res.Header.Get("Access-Control-Max-Age"), "86400")
}

func TestRejectServiceWorkerRegistration(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	req, err := http.NewRequest("GET", url+"/mysw.js", nil)
	if err != nil {
		t.Fatal(t)
	}
	req.Header.Set("Service-Worker", "script")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(t)
	}
	assert.Equal(t, res.StatusCode, 400)
	assert.Equal(t, res.Header.Get("Access-Control-Allow-Origin"), "*")
}

func TestRejectRangeAccessForNow(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	for _, method := range []string{"POST", "PUT"} {
		req, err := http.NewRequest(method, url+"/mypath", strings.NewReader("hello"))
		if err != nil {
			t.Fatal(t)
		}
		req.Header.Set("Content-Range", "bytes 2-6/100")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(t)
		}
		assert.Equal(t, res.StatusCode, 400)
		assert.Equal(t, res.Header.Get("Access-Control-Allow-Origin"), "*")
	}
}

func TestTransferSenderReceiver(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	sendBodyStr := "this is a content"
	senderReq, err := http.NewRequest("POST", url+"/mypath", strings.NewReader(sendBodyStr))
	if err != nil {
		t.Fatal(t)
	}
	senderReq.Header.Set("Content-Type", "text/plain")
	senderResCh := make(chan *http.Response)
	go func() {
		res, err := http.DefaultClient.Do(senderReq)
		if err != nil {
			t.Error(t)
			return
		}
		senderResCh <- res
	}()
	receiverReq, err := http.NewRequest("GET", url+"/mypath", nil)
	if err != nil {
		t.Fatal(t)
	}
	receiverRes, err := http.DefaultClient.Do(receiverReq)
	senderRes := <-senderResCh
	assert.Equal(t, senderRes.StatusCode, 200)
	assert.Equal(t, senderRes.Header.Get("Access-Control-Allow-Origin"), "*")

	assert.Equal(t, receiverRes.StatusCode, 200)
	assert.Equal(t, receiverRes.Header.Get("Content-Type"), "text/plain")
	assert.Equal(t, receiverRes.Header.Get("Content-Length"), strconv.Itoa(len(sendBodyStr)))
	assert.Assert(t, len(receiverRes.Header.Values("Content-Disposition")) == 0)
	assert.Equal(t, receiverRes.Header.Get("Access-Control-Allow-Origin"), "*")
	assert.Equal(t, receiverRes.Header.Get("X-Robots-Tag"), "none")
	assert.Assert(t, len(receiverRes.Header.Values("Access-Control-Expose-Headers")) == 0)
}

func TestTransferWithoutSniff(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	sendBodyStr := "this is a content"
	senderReq, err := http.NewRequest("POST", url+"/mypath", strings.NewReader(sendBodyStr))
	if err != nil {
		t.Fatal(t)
	}
	// NOTE: No content-type is specified in senderReq
	senderResCh := make(chan *http.Response)
	go func() {
		res, err := http.DefaultClient.Do(senderReq)
		if err != nil {
			t.Error(t)
			return
		}
		senderResCh <- res
	}()
	receiverReq, err := http.NewRequest("GET", url+"/mypath", nil)
	if err != nil {
		t.Fatal(t)
	}
	receiverRes, err := http.DefaultClient.Do(receiverReq)
	senderRes := <-senderResCh
	assert.Equal(t, senderRes.StatusCode, 200)

	assert.Equal(t, receiverRes.StatusCode, 200)
	assert.Assert(t, len(receiverRes.Header.Values("Content-Type")) == 0)
}

func TestTransferReceiverSender(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	receiverReq, err := http.NewRequest("GET", url+"/mypath", nil)
	if err != nil {
		t.Fatal(t)
	}
	receiverResCh := make(chan *http.Response)
	go func() {
		res, err := http.DefaultClient.Do(receiverReq)
		if err != nil {
			t.Error(t)
			return
		}
		receiverResCh <- res
	}()

	sendBodyStr := "this is a content"
	senderReq, err := http.NewRequest("POST", url+"/mypath", strings.NewReader(sendBodyStr))
	if err != nil {
		t.Fatal(t)
	}
	senderReq.Header.Set("Content-Type", "text/plain")
	senderRes, err := http.DefaultClient.Do(senderReq)
	if err != nil {
		t.Error(t)
		return
	}

	assert.Equal(t, senderRes.StatusCode, 200)
	assert.Equal(t, senderRes.Header.Get("Access-Control-Allow-Origin"), "*")

	receiverRes := <-receiverResCh
	assert.Equal(t, receiverRes.StatusCode, 200)
	assert.Equal(t, receiverRes.Header.Get("Content-Type"), "text/plain")
	assert.Equal(t, receiverRes.Header.Get("Content-Length"), strconv.Itoa(len(sendBodyStr)))
	assert.Assert(t, len(receiverRes.Header.Values("Content-Disposition")) == 0)
	assert.Equal(t, receiverRes.Header.Get("Access-Control-Allow-Origin"), "*")
	assert.Equal(t, receiverRes.Header.Get("X-Robots-Tag"), "none")
	assert.Assert(t, len(receiverRes.Header.Values("Access-Control-Expose-Headers")) == 0)
}

func TestTransferXPipingHeaders(t *testing.T) {
	server, url := serve(t)
	defer server.Shutdown(context.Background())

	sendBodyStr := "this is a content"
	senderReq, err := http.NewRequest("POST", url+"/mypath", strings.NewReader(sendBodyStr))
	if err != nil {
		t.Fatal(t)
	}
	senderReq.Header.Set("Content-Type", "text/plain")
	senderReq.Header.Add("X-Piping", "mymetadata1")
	senderReq.Header.Add("X-Piping", "mymetadata2")
	senderReq.Header.Add("X-Piping", "mymetadata3")
	senderResCh := make(chan *http.Response)
	go func() {
		res, err := http.DefaultClient.Do(senderReq)
		if err != nil {
			t.Error(t)
			return
		}
		senderResCh <- res
	}()
	receiverReq, err := http.NewRequest("GET", url+"/mypath", nil)
	if err != nil {
		t.Fatal(t)
	}
	receiverRes, err := http.DefaultClient.Do(receiverReq)
	senderRes := <-senderResCh
	assert.Equal(t, senderRes.StatusCode, 200)
	assert.Equal(t, senderRes.Header.Get("Access-Control-Allow-Origin"), "*")

	assert.Equal(t, receiverRes.StatusCode, 200)
	assert.Equal(t, receiverRes.Header.Get("Content-Type"), "text/plain")
	assert.Equal(t, receiverRes.Header.Get("Content-Length"), strconv.Itoa(len(sendBodyStr)))
	assert.Assert(t, len(receiverRes.Header.Values("Content-Disposition")) == 0)
	assert.Equal(t, receiverRes.Header.Get("Access-Control-Allow-Origin"), "*")
	assert.Equal(t, receiverRes.Header.Get("X-Robots-Tag"), "none")
	assert.Equal(t, receiverRes.Header.Get("Access-Control-Expose-Headers"), "X-Piping")
	assert.DeepEqual(t, receiverRes.Header.Values("X-Piping"), []string{"mymetadata1", "mymetadata2", "mymetadata3"})
}
