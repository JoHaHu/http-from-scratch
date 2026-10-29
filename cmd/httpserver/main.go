package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"httpServer/internal/http1"
	"httpServer/internal/mux1"
	"io"
	"net/http"
	"sync"
)

const addr = ":8080"
const tls_addr = ":8443"

func main() {

	mux := mux1.NewMux()
	mux.RegisterHandler("/metrics", mux1.DefaultMetrics)
	mux.Register("/hello", func(w http.ResponseWriter, r *http.Request) {

		fmt.Fprintf(w, "hello")
	})

	mux.Register("/{test}", func(w http.ResponseWriter, r *http.Request) {

		fmt.Fprintf(w, "hello")
	})

	mux.Register("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello index")
	})

	mux.RegisterHandler("/echo",
		mux1.NewMethodMux().
			Post(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(w, r.Body)
			}).
			Get(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "echo")
			}),
	)
	mux.Register("/upper", func(w http.ResponseWriter, r *http.Request) {
		io.Copy(UpperWriter{w}, r.Body)
	})

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(2)

	go spawn_http(&wg, errCh, mux)
	go spawn_https(&wg, errCh, mux)

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			fmt.Println("Error:", err)
		}
	}

}

func spawn_http(wg *sync.WaitGroup, errCh chan<- error, mux *mux1.PathMux) {
	defer wg.Done()
	if err := http1.ListenAndServe(addr, mux); err != nil {
		errCh <- err
	}
}

func spawn_https(wg *sync.WaitGroup, errCh chan<- error, mux *mux1.PathMux) {
	defer wg.Done()

	cert, err := tls.LoadX509KeyPair("cert.crt", "key.key")
	if err != nil {
		errCh <- err
		return
	}

	if err := http1.ListenAndServeTLS(tls_addr, mux, cert); err != nil {
		errCh <- err
	}

}

type UpperWriter struct {
	Writer io.Writer
}

func (u UpperWriter) Write(p []byte) (int, error) {
	upper := bytes.ToUpper(p)
	return u.Writer.Write(upper)
}
