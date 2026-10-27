package http1

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

type Response struct {
	conn        net.Conn
	Writer      *bufio.Writer
	GzipWriter  *gzip.Writer
	BodyWriter  io.WriteCloser
	Headers     http.Header
	headersSent bool
}

func NewResponse(conn net.Conn) *Response {

	writer := bufio.NewWriter(conn)

	bodyWriter := httputil.NewChunkedWriter(writer)

	resp := Response{
		conn:       conn,
		Writer:     writer,
		BodyWriter: bodyWriter,

		Headers:     http.Header{},
		headersSent: false,
	}
	resp.Headers.Set("Transfer-Encoding", "chunked")
	resp.Headers.Set("Server", "myttp/0.0.0")
	resp.Headers.Set("Date", time.Now().Format(time.RFC1123Z))
	resp.Headers.Set("Connection", "close")

	return &resp
}

func (r *Response) Close() error {

	err := r.BodyWriter.Close()
	if err != nil {
		panic(err)
	}
	r.Writer.WriteString("\r\n")
	err = r.Writer.Flush()
	if err != nil {
		panic(err)
	}
	err = r.conn.Close()
	if err != nil {
		panic(err)
	}

	return nil
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
		r.headersSent = true
		fmt.Fprintf(r.Writer, "HTTP/1.1 %d \r\n", statusCode)

		for k, v := range r.Headers {
			for _, vv := range v {
				fmt.Fprintf(r.Writer, "%s: %s\r\n", k, vv)
			}
		}
		fmt.Fprintf(r.Writer, "\r\n")
	}
}
