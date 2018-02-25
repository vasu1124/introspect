package mandelbrot

import "github.com/prometheus/client_golang/prometheus"

const (
	namespace = "mandelbrot"
	subsystem = "http"
)

var (
	requestCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request_count_total",
		Help:      "Counter of HTTP requests made.",
	}, []string{"proto"})

	requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request_duration_milliseconds",
		Help:      "Histogram of the time (in milliseconds) each request took.",
		Buckets:   prometheus.ExponentialBuckets(125, 2.0, 8),
	}, []string{"proto"})
)
