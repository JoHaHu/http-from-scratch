package http1

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
)

var (
	NotSupported   = errors.New("not supported")
	BadRequest     = errors.New("bad request")
	HeaderTooLarge = errors.New("header too large")
)

type ServerError struct {
	Err error
}

type ClientError struct {
	Err error
}

func (e ServerError) Error() string {
	return fmt.Sprintf("server error: %v", e.Err)
}

func (e ServerError) Unwrap() error {
	return e.Err
}

func (e ClientError) Error() string {
	return fmt.Sprintf("bad request: %v", e.Err)
}
func (e ClientError) Unwrap() error {
	return e.Err
}

func ListenAndServeTLS(addr string, handler http.Handler, cert tls.Certificate) error {
	listener, err := net.Listen("tcp", addr)
	tlsListener := tls.NewListener(listener, &tls.Config{
		Certificates: []tls.Certificate{cert},
	})

	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer tlsListener.Close()

	if handler == nil {
		handler = http.DefaultServeMux
	}

	for {
		conn, err := tlsListener.Accept()
		if err != nil {
			fmt.Errorf("accept: %w", err)
		}
		go HandleConnection(conn, handler)
	}
}

func ListenAndServe(addr string, handler http.Handler) error {
	listener, err := net.Listen("tcp", addr)

	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	if handler == nil {
		handler = http.DefaultServeMux
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Errorf("accept: %w", err)
		}
		go HandleConnection(conn, handler)
	}
}

func HandleConnection(conn net.Conn, mux http.Handler) error {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		closeConn := func() bool {

			req, err := ParseRequest(reader)

			if errors.Is(err, io.EOF) {
				return true
			}
			if err == nil {
				req.Close = !keepalive(req)
			}

			resp := NewResponse(writer, req)
			defer resp.Close()

			handleError(err, resp)
			if err != nil {
				return true
			}
			handleRequest(req, resp, mux)

			return req.Close
		}()
		if closeConn {
			return nil
		}

	}
}

func handleRequest(req *http.Request, resp *Response, mux http.Handler) {
	mux.ServeHTTP(resp, req)
}

func handleError(err error, response *Response) {
	switch {
	case errors.Is(err, NotSupported):
		response.WriteHeader(http.StatusHTTPVersionNotSupported)
	case errors.Is(err, BadRequest):

		response.WriteHeader(http.StatusBadRequest)

	case errors.Is(err, HeaderTooLarge):
		response.WriteHeader(http.StatusRequestHeaderFieldsTooLarge)
	}
}

func keepalive(req *http.Request) bool {

	switch {
	case req.Header.Get("Connection") == "close":
		return false
	case req.Header.Get("Connection") == "keep-alive":
		return true
	case req.ProtoMajor == 1 && req.ProtoMinor == 1:
		return true
	}
	return false
}
