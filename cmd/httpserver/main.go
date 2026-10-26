package main

import (
	"crypto/tls"
	"fmt"
	"httpServer/internal/http1"
	"net/http"
	"sync"
)

const addr = ":8080"
const tls_addr = ":8443"

func main() {

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello\n")
		fmt.Fprintf(w, "hello\n")
	})

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(2)

	go spawn_http(&wg, errCh)
	go spawn_https(&wg, errCh)

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			fmt.Println("Error:", err)
		}
	}

}

func spawn_http(wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()
	if err := http1.ListenAndServe(addr, nil); err != nil {
		errCh <- err
	}
}

func spawn_https(wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()

	cert, err := tls.LoadX509KeyPair("cert.crt", "key.key")
	if err != nil {
		errCh <- err
		return
	}

	if err := http1.ListenAndServeTLS(tls_addr, nil, cert); err != nil {
		errCh <- err
	}

}
