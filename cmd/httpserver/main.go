package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"httpServer/internal/http1"
	"httpServer/internal/mux1"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const addr = ":8080"
const tls_addr = ":8443"

func main() {

	innerLogger := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(&mux1.TraceLogHandler{innerLogger})
	slog.SetDefault(logger)

	ctx := context.Background()

	tp, _ := initTracer(ctx)

	defer tp.Shutdown(ctx)

	mux := mux1.NewMux()
	mux.Register("/metrics", func(w http.ResponseWriter, req *http.Request) {
		metrics.WritePrometheus(w, false)
	})

	mux.Register("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello\n")
		NormalDelay(100*time.Microsecond, 50)
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
	var handler http.Handler = mux1.RequestMonitor{mux}
	handler = &mux1.LogHandler{handler}
	handler = &mux1.TraceHandler{handler}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(2)

	go spawn_http(&wg, errCh, handler)
	go spawn_https(&wg, errCh, handler)

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			fmt.Println("Error:", err)
		}
	}

}

func spawn_http(wg *sync.WaitGroup, errCh chan<- error, mux http.Handler) {
	defer wg.Done()
	if err := http1.ListenAndServe(addr, mux); err != nil {
		errCh <- err
	}
}

func spawn_https(wg *sync.WaitGroup, errCh chan<- error, mux http.Handler) {
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

func NormalDelay(mean, stdDev time.Duration) {
	sample := float64(mean) + rand.NormFloat64()*float64(stdDev)

	if sample < 0 {
		sample = 0
	}

	time.Sleep(time.Duration(sample))
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("vtsingle-vmks-victoria-metrics-k8s-stack:10428"),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithURLPath("/insert/opentelemetry/v1/traces"),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("my-http-service"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}
