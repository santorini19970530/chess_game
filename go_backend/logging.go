// CM3070 FP code
// logging.go - handles request logging and status reporting

package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

// statusRecorder captures response status codes
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader stores and writes the HTTP status code
func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// Write writes the response body and defaults status to 200
func (sr *statusRecorder) Write(data []byte) (int, error) {
	if sr.statusCode == 0 {
		sr.statusCode = http.StatusOK
	}

	return sr.ResponseWriter.Write(data)
}

// Hijack implements http.Hijacker so WebSocket upgrades work through the recorder.
func (sr *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := sr.ResponseWriter.(interface {
		Hijack() (net.Conn, *bufio.ReadWriter, error)
	})
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
	}
	return hijacker.Hijack()
}

// statusReport maps an HTTP status code to a short label
func statusReport(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "success"
	case code >= 300 && code < 400:
		return "redirect"
	case code >= 400 && code < 500:
		return "client error"
	case code >= 500:
		return "server error"
	default:
		return "unknown"
	}
}

// withRequestLogging logs method, path, and status for each request in the server
func withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		if strings.HasPrefix(r.URL.Path, "/.well-known/") {
			return
		}

		if recorder.statusCode == 0 {
			recorder.statusCode = http.StatusOK
		}

		log.Printf("loading page: %s %s -> %d %s [%s]",
			r.Method,
			r.URL.Path,
			recorder.statusCode,
			http.StatusText(recorder.statusCode),
			statusReport(recorder.statusCode),
		)
	})
}
