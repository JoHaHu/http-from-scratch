package mux1

import (
	"net/http"
)

type Metrics struct{}

var DefaultMetrics = &Metrics{}

func (m *Metrics) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

}
