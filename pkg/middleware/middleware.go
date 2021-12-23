package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/vasu1124/introspect/pkg/logger"
)

// RequestLogger .
type RequestLogger struct {
	http.ResponseWriter

	elapsedTime   time.Duration
	forwardedFor  string
	method        string
	proto         string
	remoteAddr    string
	responseBytes int64
	requestURI    string
	time          time.Time
	status        int
	userAgent     string
}

// Log .
func (r *RequestLogger) Log() {
	timestamp := r.time.Format("02/Jan/2006 03:04:05")
	requestLine := fmt.Sprintf("%s %s %s", r.method, r.requestURI, r.proto)
	// fmt.Fprintf(os.Stdout, "%s - [%s] \"%s %d %d\" %s %f\n",
	// 	r.remoteAddr, timestamp, requestLine, r.status,
	// 	r.responseBytes, r.userAgent, r.elapsedTime.Seconds())
	logger.Log.Info("[middleware] request",
		"remoteAddr", r.remoteAddr,
		"timestamp", timestamp,
		"requestLine", requestLine,
		"status", r.status,
		"responseBytes", r.responseBytes,
		"elapsedSeconds", r.elapsedTime.Seconds())
}

func (r *RequestLogger) Write(p []byte) (int, error) {
	n, err := r.ResponseWriter.Write(p)
	r.responseBytes += int64(n)
	return n, err
}

// WriteHeader .
func (r *RequestLogger) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// RequestLoggerHandler .
type RequestLoggerHandler struct {
	handler http.Handler
}

// NewRequestLoggerHandler .
func NewRequestLoggerHandler(handler http.Handler) http.Handler {
	return &RequestLoggerHandler{
		handler: handler,
	}
}

func (h *RequestLoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rl := &RequestLogger{
		ResponseWriter: w,
		elapsedTime:    time.Duration(0),
		forwardedFor:   r.Header.Get("X-Forwarded-For"),
		method:         r.Method,
		proto:          r.Proto,
		remoteAddr:     r.RemoteAddr,
		requestURI:     r.RequestURI,
		time:           time.Time{},
		status:         http.StatusOK,
		userAgent:      r.UserAgent(),
	}

	startTime := time.Now()
	h.handler.ServeHTTP(rl, r)
	finishTime := time.Now()

	rl.time = finishTime.UTC()
	rl.elapsedTime = finishTime.Sub(startTime)
	rl.Log()
}
