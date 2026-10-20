package healthz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/olahol/melody"
	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/handler"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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
	Type            string       `json:"type"` // "init", "log", "status"
	Log             *ProbeLog    `json:"log,omitempty"`
	Logs            []ProbeLog   `json:"logs,omitempty"`
	LivenessStatus  int          `json:"livenessStatus"`
	ReadinessStatus int          `json:"readinessStatus"`
	ProbeConfig     *ProbeConfig `json:"probeConfig,omitempty"`
}

// ProbeSpec holds the full Kubernetes probe definition the UI displays/
// mirrors. Timings drive the timers; the rest feeds the manifest reference
// widget.
type ProbeSpec struct {
	Path                      string `json:"path"`
	Port                      int    `json:"port"`
	Scheme                    string `json:"scheme"`
	InitialDelaySeconds       int32  `json:"initialDelaySeconds"`
	PeriodSeconds             int32  `json:"periodSeconds"`
	TimeoutSeconds            int32  `json:"timeoutSeconds"`
	SuccessThreshold          int32  `json:"successThreshold"`
	FailureThreshold          int32  `json:"failureThreshold"`
	TerminationGracePeriodSet bool   `json:"terminationGracePeriodSet"`
	TerminationGracePeriod    int64  `json:"terminationGracePeriodSeconds"`
}

// ProbeConfig holds both probe definitions, read from the pod spec, plus the
// rendered YAML used by the configuration reference widget.
type ProbeConfig struct {
	PodName   string    `json:"podName,omitempty"`
	Namespace string    `json:"namespace,omitempty"`
	Container string    `json:"container,omitempty"`
	Liveness  ProbeSpec `json:"liveness"`
	Readiness ProbeSpec `json:"readiness"`
	Source    string    `json:"source"` // "pod" or "default"
	YAML      string    `json:"yaml"`
}

// defaultProbeConfig mirrors kubernetes/introspect/deployment.yaml.
func defaultProbeConfig() *ProbeConfig {
	cfg := &ProbeConfig{
		Liveness: ProbeSpec{
			Path: "/healthz", Port: 9090, Scheme: "HTTPS",
			InitialDelaySeconds: 15, PeriodSeconds: 20, TimeoutSeconds: 2,
			SuccessThreshold: 1, FailureThreshold: 3,
		},
		Readiness: ProbeSpec{
			Path: "/healthzr", Port: 9090, Scheme: "HTTPS",
			InitialDelaySeconds: 16, PeriodSeconds: 15, TimeoutSeconds: 3,
			SuccessThreshold: 1, FailureThreshold: 4,
		},
		Source: "default",
	}
	cfg.YAML = renderProbeYAML(cfg)
	return cfg
}

// renderProbeYAML renders the effective probe configuration as it appears in
// the pod manifest container spec.
func renderProbeYAML(pc *ProbeConfig) string {
	probe := func(name string, p ProbeSpec) string {
		var sb strings.Builder
		fmt.Fprintf(&sb, "        %s:\n", name)
		fmt.Fprintf(&sb, "          httpGet:\n")
		fmt.Fprintf(&sb, "            path: %s\n", p.Path)
		fmt.Fprintf(&sb, "            port: %d\n", p.Port)
		fmt.Fprintf(&sb, "            scheme: %s\n", p.Scheme)
		fmt.Fprintf(&sb, "          initialDelaySeconds: %d\n", p.InitialDelaySeconds)
		fmt.Fprintf(&sb, "          periodSeconds: %d\n", p.PeriodSeconds)
		fmt.Fprintf(&sb, "          timeoutSeconds: %d\n", p.TimeoutSeconds)
		fmt.Fprintf(&sb, "          successThreshold: %d\n", p.SuccessThreshold)
		fmt.Fprintf(&sb, "          failureThreshold: %d\n", p.FailureThreshold)
		if p.TerminationGracePeriodSet {
			fmt.Fprintf(&sb, "          terminationGracePeriodSeconds: %d\n", p.TerminationGracePeriod)
		}
		return sb.String()
	}

	var sb strings.Builder
	if pc.PodName != "" {
		fmt.Fprintf(&sb, "# live spec from pod %s", pc.PodName)
		if pc.Namespace != "" {
			fmt.Fprintf(&sb, " (namespace %s)", pc.Namespace)
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("# fallback defaults \u2014 live pod manifest unavailable\n")
	}
	fmt.Fprintf(&sb, "      containers:\n")
	name := pc.Container
	if name == "" {
		name = "introspect"
	}
	fmt.Fprintf(&sb, "      - name: %s\n", name)
	sb.WriteString(probe("livenessProbe", pc.Liveness))
	sb.WriteString(probe("readinessProbe", pc.Readiness))
	return sb.String()
}

// Handler implements handler.Handler and handler.Closer for the healthz endpoints.
type Handler struct {
	mu             sync.RWMutex
	statusl        int
	statusr        int
	logs           []ProbeLog
	melody         *melody.Melody
	unsubscribeLog func()
	probeConfig    *ProbeConfig
}

// New creates a new healthz handler.
func New() *Handler {
	h := &Handler{
		statusl:     http.StatusOK,
		statusr:     http.StatusOK,
		logs:        make([]ProbeLog, 0, 100),
		melody:      melody.New(),
		probeConfig: defaultProbeConfig(),
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
			ProbeConfig:     h.getProbeConfig(),
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

	// Best-effort: resolve probe timing from this pod's manifest so the UI
	// reflects the real livenessProbe/readinessProbe configuration.
	h.loadProbeConfig(ctx)

	logger.Log.Info("[healthz] registered /healthz, /healthzr, /healthz/ui, and /healthzws")
}

// getProbeConfig returns the current probe timing config (thread-safe).
func (h *Handler) getProbeConfig() *ProbeConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	cfg := *h.probeConfig
	return &cfg
}

// loadProbeConfig reads periodSeconds/timeoutSeconds/failureThreshold of the
// liveness and readiness probes from this pod's own spec (i.e. the manifest
// the kubelet actually uses). Falls back to the deployment.yaml defaults when
// the API server is unreachable or the pod spec has no probes defined.
func (h *Handler) loadProbeConfig(ctx context.Context) {
	rc, err := config.GetConfig()
	if err != nil {
		logger.Log.Info("[healthz] no kubeconfig available; using deployment.yaml probe defaults", "error", err.Error())
		return
	}
	clientset, err := kubernetes.NewForConfig(rc)
	if err != nil {
		logger.Log.Error(err, "[healthz] failed to create clientset; using probe defaults")
		return
	}

	podName, err := os.Hostname()
	if err != nil || podName == "" {
		logger.Log.Error(err, "[healthz] unable to determine hostname; using probe defaults")
		return
	}
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		if b, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
			namespace = strings.TrimSpace(string(b))
		}
	}
	if namespace == "" {
		namespace = "default"
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		logger.Log.Error(err, "[healthz] unable to read own pod manifest; using probe defaults",
			"namespace", namespace, "pod", podName)
		return
	}

	pc := defaultProbeConfig()
	pc.Source = "pod"
	pc.PodName = podName
	pc.Namespace = namespace

	var container *corev1.Container
	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]
		if c.Name == "introspect" {
			container = c
			break
		}
	}
	if container == nil && len(pod.Spec.Containers) > 0 {
		container = &pod.Spec.Containers[0]
	}
	if container == nil {
		logger.Log.Info("[healthz] pod spec has no containers; using probe defaults", "pod", podName)
		return
	}
	pc.Container = container.Name

	// applyProbe copies the effective probe definition from the pod spec,
	// resolving Kubernetes defaults where fields are omitted.
	applyProbe := func(dst *ProbeSpec, p *corev1.Probe, path string, port int) {
		if p == nil {
			return
		}
		dst.Path = path
		dst.Port = port
		dst.Scheme = "HTTP"
		dst.InitialDelaySeconds = p.InitialDelaySeconds
		if p.HTTPGet != nil {
			if p.HTTPGet.Path != "" {
				dst.Path = p.HTTPGet.Path
			}
			if p.HTTPGet.Port.Type == intstr.Int && p.HTTPGet.Port.IntVal > 0 {
				dst.Port = int(p.HTTPGet.Port.IntVal)
			} else if p.HTTPGet.Port.Type == intstr.String {
				dst.Port = port // named port, keep default in display
			}
			if string(p.HTTPGet.Scheme) != "" {
				dst.Scheme = string(p.HTTPGet.Scheme)
			}
		}
		if p.PeriodSeconds > 0 {
			dst.PeriodSeconds = p.PeriodSeconds
		} else {
			dst.PeriodSeconds = 10 // k8s default
		}
		if p.TimeoutSeconds > 0 {
			dst.TimeoutSeconds = p.TimeoutSeconds
		} else {
			dst.TimeoutSeconds = 1 // k8s default
		}
		if p.SuccessThreshold > 0 {
			dst.SuccessThreshold = p.SuccessThreshold
		} else {
			dst.SuccessThreshold = 1 // k8s default
		}
		if p.FailureThreshold > 0 {
			dst.FailureThreshold = p.FailureThreshold
		} else {
			dst.FailureThreshold = 3 // k8s default
		}
		if p.TerminationGracePeriodSeconds != nil {
			dst.TerminationGracePeriodSet = true
			dst.TerminationGracePeriod = *p.TerminationGracePeriodSeconds
		}
	}
	applyProbe(&pc.Liveness, container.LivenessProbe, "/healthz", 9090)
	applyProbe(&pc.Readiness, container.ReadinessProbe, "/healthzr", 9090)
	pc.YAML = renderProbeYAML(pc)

	h.mu.Lock()
	h.probeConfig = pc
	h.mu.Unlock()

	logger.Log.Info("[healthz] probe config loaded from pod manifest",
		"pod", podName, "namespace", namespace, "container", container.Name,
		"livenessPeriod", pc.Liveness.PeriodSeconds,
		"livenessTimeout", pc.Liveness.TimeoutSeconds,
		"livenessFailureThreshold", pc.Liveness.FailureThreshold,
		"readinessPeriod", pc.Readiness.PeriodSeconds,
		"readinessTimeout", pc.Readiness.TimeoutSeconds,
		"readinessFailureThreshold", pc.Readiness.FailureThreshold)
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
