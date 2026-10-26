package http1

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
)

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

func HandleConnection(conn net.Conn, mux http.Handler) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	req, err := ParseRequest(reader)
	if err != nil {
		fmt.Errorf("parse request: %w", err)
	}
	writer := NewResponse(conn)
	mux.ServeHTTP(writer, req)
}
