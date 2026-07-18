package mandelbrot

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math"
	"math/cmplx"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
	"github.com/vasu1124/introspect/pkg/handler"
)

var (
	requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mandelbrot_requests_total",
			Help: "Total number of requests to mandelbrot endpoint",
		},
		[]string{"proto"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mandelbrot_request_duration_seconds",
			Help:    "Request duration for mandelbrot endpoint",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"proto"},
	)
)

func init() {
	prometheus.MustRegister(requestCount, requestDuration)
}

// Handler implements server.Handler for the mandelbrot endpoint.
type Handler struct{}

// New creates a new mandelbrot handler.
func New() *Handler {
	return &Handler{}
}

// Name implements server.Handler.
func (h *Handler) Name() string {
	return "mandelbrot"
}

// RegisterRoutes implements server.Handler.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, ctx context.Context) {
	mux.HandleFunc("/mandelbrot/view", h.ServeView)
	mux.Handle("/mandelbrot", h)
	logger.Log.Info("[mandelbrot] registered /mandelbrot/view and /mandelbrot")
}

// ServeHTTP handles the mandelbrot image generation.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds() * 1e3
		proto := strconv.Itoa(r.ProtoMajor) + "." + strconv.Itoa(r.ProtoMinor)
		requestCount.WithLabelValues(proto).Inc()
		requestDuration.WithLabelValues(proto).Observe(duration)
	}()

	if err := r.ParseForm(); err != nil {
		logger.Log.Error(err, "[mandelbrot] ParseForm error")
	}

	var xmin, ymin, xmax, ymax float64
	xmin = form2float64(r.Form["xmin"], -1.8)
	ymin = form2float64(r.Form["ymin"], -1.5)
	xmax = form2float64(r.Form["xmax"], 1.2)
	ymax = form2float64(r.Form["ymax"], 1.5)

	if r.Form["steps"] == nil {
		img := mandelbrot(xmin, ymin, xmax, ymax)
		png.Encode(w, img)
	} else {
		var steps, xfmin, yfmin, xfmax, yfmax float64
		xfmin = form2float64(r.Form["xfmin"], -1.110)
		yfmin = form2float64(r.Form["yfmin"], 0.228)
		xfmax = form2float64(r.Form["xfmax"], -1.106)
		yfmax = form2float64(r.Form["yfmax"], 0.232)
		steps = form2float64(r.Form["steps"], 10)

		var images []*image.Paletted
		var delays []int
		for i := 0.0; i <= steps; i += 1.0 {
			img := mandelbrot(
				xmin+(xfmin-xmin)*math.Tanh(4*i/steps),
				ymin+(yfmin-ymin)*math.Tanh(4*i/steps),
				xmax+(xfmax-xmax)*math.Tanh(4*i/steps),
				ymax+(yfmax-ymax)*math.Tanh(4*i/steps))
			var buf bytes.Buffer
			var opt gif.Options
			opt.NumColors = 256

			gif.Encode(&buf, img, &opt)
			g, _ := gif.DecodeAll(&buf)
			images = append(images, g.Image[0])
			delays = append(delays, 50)
		}

		gif.EncodeAll(w, &gif.GIF{
			Image: images,
			Delay: delays,
		})
	}
}

// ServeView serves the mandelbrot UI page.
func (h *Handler) ServeView(w http.ResponseWriter, r *http.Request) {
	data := struct {
		assets.CommonData
	}{
		CommonData: assets.CommonData{Version: version.Version, Flag: version.Flag},
	}

	if err := assets.ExecuteTemplate(w, "mandelbrot.html", data); err != nil {
		logger.Log.Error(err, "[mandelbrot] executing template")
	}
}

func form2float64(form []string, def float64) (f float64) {
	f = def
	if form != nil {
		f, _ = strconv.ParseFloat(form[0], 64)
	}
	return
}

func mandelbrot(xmin, ymin, xmax, ymax float64) image.Image {
	const (
		width, height = 512, 512
	)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for py := 0; py < height; py++ {
		y := float64(py)/height*(ymax-ymin) + ymin
		for px := 0; px < width; px++ {
			x := float64(px)/width*(xmax-xmin) + xmin
			z := complex(x, y)
			img.Set(px, py, m(z))
		}
	}

	return img
}

func m(z complex128) color.Color {
	const (
		iterations = 200
		contrast   = 15
	)

	var v complex128
	for n := uint8(0); n < iterations; n++ {
		v = v*v + z
		if cmplx.Abs(v) > 2 {
			r, g, b := color.YCbCrToRGB(255, 255-contrast*n, 255-contrast*n)
			return color.RGBA{r, g, b, 255}
		}
	}
	return color.Black
}

// Ensure Handler implements handler.Handler.
var _ handler.Handler = (*Handler)(nil)