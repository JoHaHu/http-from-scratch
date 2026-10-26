package http1

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

type Response struct {
	Writer      *bufio.Writer
	Headers     http.Header
	headersSent bool
}

func NewResponse(conn net.Conn) *Response {
	writer := bufio.NewWriter(conn)
	resp := Response{
		Writer:      writer,
		Headers:     http.Header{},
		headersSent: false,
	}
	return &resp
}

func (r *Response) Header() http.Header {
	return r.Headers
}

func (r *Response) Write(data []byte) (n int, err error) {

	if !r.headersSent {
		r.WriteHeader(http.StatusOK)
	}
	n, err = r.Writer.Write(data)
	if err != nil {
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
}
