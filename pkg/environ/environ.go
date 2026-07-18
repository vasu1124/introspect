package environ

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/handler"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/network"
	"github.com/vasu1124/introspect/pkg/osinfo"
	"github.com/vasu1124/introspect/pkg/version"
)

var count uint32

var (
	requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "environ_requests_total",
			Help: "Total number of requests to environ endpoint",
		},
		[]string{"proto"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "environ_request_duration_seconds",
			Help:    "Duration of environ requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"proto"},
	)
)

func init() {
	prometheus.MustRegister(requestCount)
	prometheus.MustRegister(requestDuration)
}

// Handler implements server.Handler for the environ endpoint.
type Handler struct{}

// New creates a new environ handler.
func New() *Handler {
	return &Handler{}
}

// Name implements server.Handler.
func (h *Handler) Name() string {
	return "environ"
}

// RegisterRoutes implements server.Handler.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, ctx context.Context) {
	mux.HandleFunc("/environ", h.ServeHTTP)
	logger.Log.Info("[environ] registered /environ")
}

// ServeHTTP handles the environ request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	atomic.AddUint32(&count, 1)

	start := time.Now()
	h.serveEnviron(w, r)
	duration := time.Since(start).Seconds()

	proto := strconv.Itoa(r.ProtoMajor) + "." + strconv.Itoa(r.ProtoMinor)
	requestCount.WithLabelValues(proto).Inc()
	requestDuration.WithLabelValues(proto).Observe(duration)
}

func (h *Handler) serveEnviron(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logger.Log.Error(err, "[environ] ParseForm error")
	}

	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) == 2 {
			envMap[kv[0]] = kv[1]
		}
	}

	requestMap := map[string][]string{
		"ContentLength":  {strconv.FormatInt(r.ContentLength, 10)},
		"Host":           {r.Host},
		"RemoteAddr":     {r.RemoteAddr},
		"Proto":          {r.Proto},
		"Method":         {r.Method},
		"Referer":        {r.Referer()},
		"RequestURI":     {r.RequestURI},
		"TransferEncoding": r.TransferEncoding,
	}

	processMap := map[string]string{
		"GO ARCH":           runtime.GOARCH,
		"GO OS":             runtime.GOOS,
		"GO Version":        runtime.Version(),
		"GO NumCPU":         strconv.Itoa(runtime.NumCPU()),
		"GO NumGoroutine":   strconv.Itoa(runtime.NumGoroutine()),
		"Introspect Version": version.Get().GitVersion,
		"Introspect TreeState": version.Get().GitTreeState,
		"Introspect Commit": version.Get().GitCommit,
		"Introspect BuildDate": version.Get().BuildDate,
	}

	serverMap := map[string]string{
		"Machine Architecture": osinfo.Utsname.Machine,
		"Nodename":             osinfo.Utsname.Nodename,
	}

	data := struct {
		assets.CommonData
		Environment map[string]string
		Header      map[string][]string
		Form        map[string][]string
		Request     map[string][]string
		Process     map[string]string
		OS          map[string]string
		Server      map[string]string
		Counter     uint32
		Network     network.Data
	}{
		CommonData: assets.CommonData{Version: version.Version, Flag: version.Flag},
		Environment: envMap,
		Header:      r.Header,
		Form:        r.Form,
		Request:     requestMap,
		Process:     processMap,
		OS:          osinfo.OSrelease,
		Server:      serverMap,
		Counter:     count,
		Network:     network.NetworkData,
	}

	if err := assets.ExecuteTemplate(w, "environ.html", data); err != nil {
		logger.Log.Error(err, "[environ] executing template")
		fmt.Fprint(w, "[environ] executing template: ", err)
	}
}

// Ensure Handler implements handler.Handler.
var _ handler.Handler = (*Handler)(nil)