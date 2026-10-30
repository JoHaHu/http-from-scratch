package mux1

import (
	"context"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

type TraceHandler struct {
	http.Handler
}

func (t *TraceHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	propagator := otel.GetTextMapPropagator()
	ctx := propagator.Extract(req.Context(), propagation.HeaderCarrier(req.Header))

	tracer := otel.Tracer("http-server")
	ctx, span := tracer.Start(
		ctx,
		req.Method+" "+req.URL.Path,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.HTTPRequestMethodKey.String(req.Method),
			semconv.URLPath(req.URL.Path),
		),
	)
	defer span.End()

	t.Handler.ServeHTTP(w, req.WithContext(ctx))

}

type TraceLogHandler struct {
	slog.Handler
}

func (h *TraceLogHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		r.AddAttrs(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}
