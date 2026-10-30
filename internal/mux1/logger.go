package mux1

import (
	"log/slog"
	"net/http"
)

type LogHandler struct {
	http.Handler
}

func (l *LogHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	slog.InfoContext(req.Context(), "http_request", "method", req.Method, "path", req.URL.EscapedPath())
	l.Handler.ServeHTTP(w, req)
}
