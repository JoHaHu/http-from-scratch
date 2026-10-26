package http1

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Response struct {
	conn        net.Conn
	Writer      *bufio.Writer
	Headers     http.Header
	headersSent bool
}

func NewResponse(conn net.Conn) *Response {
	writer := bufio.NewWriter(conn)
	resp := Response{
		conn:        conn,
		Writer:      writer,
		Headers:     http.Header{},
		headersSent: false,
	}
	resp.Headers.Set("Connection", "close")
	resp.Headers.Set("Transfer-Encoding", "chunked")
	resp.Headers.Set("Server", "myttp/0.0.0")
	resp.Headers.Set("Date", time.Now().Format(time.RFC1123Z))

	return &resp
}

func (r *Response) Close() error {
	r.Writer.Write([]byte("0\r\n\r\n"))
	r.Writer.Flush()
	r.conn.Close()
	return nil
}

func (r *Response) Header() http.Header {
	return r.Headers
}

func (r *Response) Write(data []byte) (n int, err error) {

	if !r.headersSent {
		r.WriteHeader(http.StatusOK)
	}

	length := len(data)
	hexStr := strconv.FormatInt(int64(length), 16)

	written, err := r.Writer.Write([]byte(hexStr))
	if err != nil {
		n += written
		return n, err
	}

	written, err = r.Writer.Write([]byte("\r\n"))
	if err != nil {
		n += written
		return n, err
	}

	written, err = r.Writer.Write(data)
	if err != nil {
		n += written
		return n, err
	}

	written, err = r.Writer.Write([]byte("\r\n"))
	if err != nil {
		n += written
		return n, err
	}
	err = r.Writer.Flush()
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
	r.Writer.Flush()
}
