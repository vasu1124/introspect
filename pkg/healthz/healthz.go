package healthz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/olahol/melody"
	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/handler"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
)

// ProbeLog represents a structured log entry for a liveness or readiness probe event.
type ProbeLog struct {
	ID         int64  `json:"id"`
	Timestamp  string `json:"timestamp"`
	Type       string `json:"type"`       // "liveness" or "readiness"
	Probe      string `json:"probe"`      // "Liveness (/healthz)" or "Readiness (/healthzr)"
	Method     string `json:"method"`     // "GET", "LOG", etc.
	Path       string `json:"path"`       // "/healthz" or "/healthzr"
	Query      string `json:"query"`      // e.g. "live", "die", or ""
	Status     int    `json:"status"`     // 200, 500, 503
	StatusText string `json:"statusText"` // "OK", "Internal Server Error", "Service Unavailable"
	RemoteIP   string `json:"remoteIp"`   // client IP/addr
	UserAgent  string `json:"userAgent"`  // kube-probe/1.28, curl, browser...
	Duration   string `json:"duration"`   // latency string
	Message    string `json:"message"`    // log message text
	Level      string `json:"level"`      // "INFO", "WARN", "ERROR"
}

// WebSocketMessage represents the payload sent over WebSocket to clients.
type WebSocketMessage struct {
	Type            string     `json:"type"` // "init", "log", "status"
	Log             *ProbeLog  `json:"log,omitempty"`
	Logs            []ProbeLog `json:"logs,omitempty"`
	LivenessStatus  int        `json:"livenessStatus"`
	ReadinessStatus int        `json:"readinessStatus"`
}

// Handler implements handler.Handler and handler.Closer for the healthz endpoints.
type Handler struct {
	mu             sync.RWMutex
	statusl        int
	statusr        int
	logs           []ProbeLog
	melody         *melody.Melody
	unsubscribeLog func()
}

// New creates a new healthz handler.
func New() *Handler {
	h := &Handler{
		statusl: http.StatusOK,
		statusr: http.StatusOK,
		logs:    make([]ProbeLog, 0, 100),
		melody:  melody.New(),
	}

	h.setupMelody()
	h.setupLogSubscription()

	return h
}

func (h *Handler) setupMelody() {
	h.melody.HandleConnect(func(s *melody.Session) {
		h.mu.RLock()
		logsCopy := make([]ProbeLog, len(h.logs))
		copy(logsCopy, h.logs)
		lStatus := h.statusl
		rStatus := h.statusr
		h.mu.RUnlock()

		msg := WebSocketMessage{
			Type:            "init",
			Logs:            logsCopy,
			LivenessStatus:  lStatus,
			ReadinessStatus: rStatus,
		}
		if b, err := json.Marshal(msg); err == nil {
			_ = s.Write(b)
		}
	})
}

func (h *Handler) setupLogSubscription() {
	// Subscribe to logger.Log entries and filter specifically for liveness and readiness probe logs
	h.unsubscribeLog = logger.Subscribe(func(entry logger.LogEntry) {
		// Filter criteria: message contains [healthz] or [healthzr],
		// or path field is /healthz or /healthzr
		isHealthz := strings.Contains(entry.Message, "[healthz]")
		isHealthzr := strings.Contains(entry.Message, "[healthzr]")

		pathVal, _ := entry.Fields["path"].(string)
		if strings.HasPrefix(pathVal, "/healthz") && pathVal != "/healthz/ui" && pathVal != "/healthzws" && pathVal != "/healthz/ws" {
			if pathVal == "/healthzr" {
				isHealthzr = true
			} else {
				isHealthz = true
			}
		}

		if !isHealthz && !isHealthzr {
			// Filter out all non-probe logs (e.g. /metrics, /dynconfig, etc.)
			return
		}

		// Don't duplicate direct probe request logs that our handleHealthz/handleHealthzr already records
		if entry.Message == "[healthz]" || entry.Message == "[healthzr]" || entry.Message == "[http] request" {
			return
		}

		probeType := "liveness"
		probeLabel := "Liveness (/healthz)"
		path := "/healthz"
		if isHealthzr {
			probeType = "readiness"
			probeLabel = "Readiness (/healthzr)"
			path = "/healthzr"
		}

		level := strings.ToUpper(entry.Level)
		if level == "" {
			level = "INFO"
		}

		status := http.StatusOK
		if level == "ERROR" || level == "FATAL" || level == "PANIC" {
			status = http.StatusInternalServerError
		} else if level == "WARN" {
			status = http.StatusServiceUnavailable
		}

		logItem := ProbeLog{
			ID:         time.Now().UnixNano(),
			Timestamp:  entry.Time.Format("15:04:05.000"),
			Type:       probeType,
			Probe:      probeLabel,
			Method:     "LOG",
			Path:       path,
			Status:     status,
			StatusText: level,
			Message:    entry.Message,
			Level:      level,
		}

		h.appendAndBroadcast(logItem)
	})
}

// Name implements handler.Handler.
func (h *Handler) Name() string {
	return "healthz"
}

// RegisterRoutes implements handler.Handler.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, ctx context.Context) {
	mux.HandleFunc("/healthz", h.handleHealthz)
	mux.HandleFunc("/healthzr", h.handleHealthzr)
	mux.HandleFunc("/healthz/ui", h.handleHealthzUI)
	mux.HandleFunc("/healthzws", func(w http.ResponseWriter, r *http.Request) {
		_ = h.melody.HandleRequest(w, r)
	})
	mux.HandleFunc("/healthz/ws", func(w http.ResponseWriter, r *http.Request) {
		_ = h.melody.HandleRequest(w, r)
	})
	logger.Log.Info("[healthz] registered /healthz, /healthzr, /healthz/ui, and /healthzws")
}

// handleHealthz handles the liveness check endpoint.
// Query params: ?die (sets 500), ?live (sets 200)
func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	err := r.ParseForm()
	if err != nil {
		logger.Log.Error(err, "[healthz] ParseForm error")
	}

	h.mu.Lock()
	var action string
	if r.Form["die"] != nil {
		h.statusl = http.StatusInternalServerError
		action = "die"
	} else if r.Form["live"] != nil {
		h.statusl = http.StatusOK
		action = "live"
	}
	status := h.statusl
	h.mu.Unlock()

	logger.Log.Info("[healthz]", "status", status)

	w.WriteHeader(status)
	_, _ = w.Write(fmt.Appendf(nil, "Status: %d", status))

	msg := fmt.Sprintf("[healthz] Liveness probe check: %d %s", status, http.StatusText(status))
	if action != "" {
		msg += fmt.Sprintf(" (?%s)", action)
	}
	h.recordProbe("liveness", r.Method, r.URL.Path, r.URL.RawQuery, r.RemoteAddr, r.UserAgent(), status, time.Since(start), msg)
}

// handleHealthzr handles the readiness check endpoint.
// Query params: ?die (sets 503), ?live (sets 200)
func (h *Handler) handleHealthzr(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	err := r.ParseForm()
	if err != nil {
		logger.Log.Error(err, "[healthzr] ParseForm error")
	}

	h.mu.Lock()
	var action string
	if r.Form["die"] != nil {
		h.statusr = http.StatusServiceUnavailable
		action = "die"
	} else if r.Form["live"] != nil {
		h.statusr = http.StatusOK
		action = "live"
	}
	status := h.statusr
	h.mu.Unlock()

	logger.Log.Info("[healthzr]", "status", status)

	w.WriteHeader(status)
	_, _ = w.Write(fmt.Appendf(nil, "Status: %d", status))

	msg := fmt.Sprintf("[healthzr] Readiness probe check: %d %s", status, http.StatusText(status))
	if action != "" {
		msg += fmt.Sprintf(" (?%s)", action)
	}
	h.recordProbe("readiness", r.Method, r.URL.Path, r.URL.RawQuery, r.RemoteAddr, r.UserAgent(), status, time.Since(start), msg)
}

func (h *Handler) recordProbe(probeType string, method, path, query, remote, userAgent string, status int, duration time.Duration, msgText string) {
	level := "INFO"
	if status >= 500 {
		level = "ERROR"
	} else if status >= 400 {
		level = "WARN"
	}

	probeLabel := "Liveness (/healthz)"
	if probeType == "readiness" {
		probeLabel = "Readiness (/healthzr)"
	}

	statusText := http.StatusText(status)
	if statusText == "" {
		statusText = fmt.Sprintf("HTTP %d", status)
	}

	if userAgent == "" {
		userAgent = "unknown"
	}

	logItem := ProbeLog{
		ID:         time.Now().UnixNano(),
		Timestamp:  time.Now().Format("15:04:05.000"),
		Type:       probeType,
		Probe:      probeLabel,
		Method:     method,
		Path:       path,
		Query:      query,
		Status:     status,
		StatusText: statusText,
		RemoteIP:   remote,
		UserAgent:  userAgent,
		Duration:   duration.Round(time.Microsecond).String(),
		Message:    msgText,
		Level:      level,
	}

	h.appendAndBroadcast(logItem)
}

func (h *Handler) appendAndBroadcast(logItem ProbeLog) {
	h.mu.Lock()
	h.logs = append(h.logs, logItem)
	if len(h.logs) > 100 {
		h.logs = h.logs[len(h.logs)-100:]
	}
	lStatus := h.statusl
	rStatus := h.statusr
	h.mu.Unlock()

	msg := WebSocketMessage{
		Type:            "log",
		Log:             &logItem,
		LivenessStatus:  lStatus,
		ReadinessStatus: rStatus,
	}

	if b, err := json.Marshal(msg); err == nil {
		_ = h.melody.Broadcast(b)
	}
}

// handleHealthzUI serves the health check UI page.
func (h *Handler) handleHealthzUI(w http.ResponseWriter, r *http.Request) {
	data := struct {
		assets.CommonData
		LivenessStatus  int
		ReadinessStatus int
		RecentLogs      []ProbeLog
	}{
		CommonData:      assets.CommonData{Version: version.Version, Flag: version.Flag},
		LivenessStatus:  h.getLivenessStatus(),
		ReadinessStatus: h.getReadinessStatus(),
		RecentLogs:      h.getRecentLogs(),
	}

	if err := assets.ExecuteTemplate(w, "healthz.html", data); err != nil {
		logger.Log.Error(err, "[healthz] executing template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// getLivenessStatus returns the current liveness status (thread-safe).
func (h *Handler) getLivenessStatus() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.statusl
}

// getReadinessStatus returns the current readiness status (thread-safe).
func (h *Handler) getReadinessStatus() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.statusr
}

// getRecentLogs returns a copy of recent probe logs (thread-safe).
func (h *Handler) getRecentLogs() []ProbeLog {
	h.mu.RLock()
	defer h.mu.RUnlock()
	logsCopy := make([]ProbeLog, len(h.logs))
	copy(logsCopy, h.logs)
	return logsCopy
}

// Close implements handler.Closer.
func (h *Handler) Close() error {
	if h.unsubscribeLog != nil {
		h.unsubscribeLog()
	}
	if h.melody != nil {
		_ = h.melody.Close()
	}
	return nil
}

// Ensure Handler implements handler.Handler and handler.Closer.
var (
	_ handler.Handler = (*Handler)(nil)
	_ handler.Closer  = (*Handler)(nil)
)
