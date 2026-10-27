package http1

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"time"
)

type Response struct {
	req         *http.Request
	Writer      *bufio.Writer
	BodyWriter  io.WriteCloser
	Headers     http.Header
	headersSent bool
	keepalive   bool
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

func NopWriteCloser(w io.Writer) io.WriteCloser {
	return nopWriteCloser{w}
}

func (r *Response) Close() error {

	if !r.headersSent {
		r.WriteHeader(http.StatusOK)
	}

	err := r.BodyWriter.Close()
	if err != nil {
		return err
	}
	r.sendTrailers()

	return r.Writer.Flush()
}

func NewResponse(writer *bufio.Writer, req *http.Request) *Response {

	bodyWriter := NopWriteCloser(writer)

	resp := Response{
		req:        req,
		Writer:     writer,
		BodyWriter: bodyWriter,

		Headers:     http.Header{},
		headersSent: false,
	}

	if req != nil && !req.Close {
		resp.BodyWriter = httputil.NewChunkedWriter(writer)
		resp.Headers.Set("Connection", "keep-alive")
		resp.Headers.Set("Transfer-Encoding", "chunked")
	}

	resp.Headers.Set("Server", "myttp/0.0.0")
	resp.Headers.Set("Date", time.Now().Format(time.RFC1123Z))

	return &resp
}

func (r *Response) Header() http.Header {
	return r.Headers
}

func (r *Response) Write(data []byte) (n int, err error) {
	if !r.headersSent {
		if r.Headers.Get("Content-Type") == "" {
			ct := http.DetectContentType(data[:min(len(data), 512)])
			r.Headers.Set("Content-Type", ct)
		}
		r.WriteHeader(http.StatusOK)
	}
	n, err = r.BodyWriter.Write(data)

	return n, err
}

func (r *Response) WriteHeader(statusCode int) {

	if !r.headersSent {

		// Close request before sending results to check for misformed content
		sendHeaders := true

		if r.req != nil {
			err := r.req.Body.Close()
			if err != nil {
				statusCode = http.StatusBadRequest
				sendHeaders = false
				r.req.Close = true
			}
		}

		r.headersSent = true
		fmt.Fprintf(r.Writer, "HTTP/1.1 %d \r\n", statusCode)

		if sendHeaders {
			for k, v := range r.Headers {
				for _, vv := range v {
					fmt.Fprintf(r.Writer, "%s: %s\r\n", k, vv)
				}
			}
		}

		fmt.Fprintf(r.Writer, "\r\n")
	}
}

func (r *Response) sendTrailers() {
	if r.Headers.Get("Transfer-Encoding") == "chunked" {
		r.Writer.WriteString("X-Check: done\r\n")
		r.Writer.WriteString("\r\n")
	}
}
