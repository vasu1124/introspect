package dynconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/olahol/melody"
	"github.com/spf13/viper"
	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/handler"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

// FsEvent represents a logged filesystem notification event.
type FsEvent struct {
	ID        int64  `json:"id"`
	Timestamp string `json:"timestamp"`
	Op        string `json:"op"`
	File      string `json:"file"`
	Message   string `json:"message"`
}

// StateMessage represents a WebSocket message sent to connected clients.
type StateMessage struct {
	Type          string    `json:"type"` // "init" or "fsnotify"
	RawContent    string    `json:"rawContent"`
	ConfigJSON    string    `json:"configJson"`
	ConfigMapName string    `json:"configMapName,omitempty"`
	Namespace     string    `json:"namespace,omitempty"`
	InCluster     bool      `json:"inCluster"`
	Event         *FsEvent  `json:"event,omitempty"`
	Events        []FsEvent `json:"events,omitempty"`
}

// Handler implements handler.Handler for the dynamic config endpoint.
type Handler struct {
	mu            sync.RWMutex
	example       map[string]any
	rawContent    string
	eventLog      []FsEvent
	ctx           context.Context
	cancelCtx     context.CancelFunc
	watcher       *fsnotify.Watcher
	config        *viper.Viper
	kubeClient    clientset.Interface
	configMapName string
	namespace     string
	localFilePath string
	melody        *melody.Melody
}

// New creates a new dynconfig handler.
func New() *Handler {
	return NewWithClient(nil)
}

// NewWithClient creates a new dynconfig handler with an optional kubernetes client.
func NewWithClient(client clientset.Interface) *Handler {
	h := &Handler{
		example:       make(map[string]any),
		eventLog:      make([]FsEvent, 0),
		configMapName: "introspect-dynconfig",
		namespace:     "default",
		melody:        melody.New(),
	}

	if customCM := os.Getenv("DYNCONFIG_CONFIGMAP_NAME"); customCM != "" {
		h.configMapName = customCM
	}
	if ns, ok := os.LookupEnv("NAMESPACE"); ok && ns != "" {
		h.namespace = ns
	}

	if client != nil {
		h.kubeClient = client
	} else {
		h.kubeClient = initKubeClient()
	}

	v := viper.New()
	v.SetConfigName("example")
	v.AddConfigPath("/etc/config/dynconfig")
	v.AddConfigPath("./config/dynconfig")
	v.AddConfigPath("./etc/config/dynconfig")
	v.AddConfigPath("./etc/config")
	v.AddConfigPath("/etc/config")

	if err := v.ReadInConfig(); err != nil {
		logger.Log.Info("[dynconfig] Config file not found initially", "error", err)
	}

	if configFile := v.ConfigFileUsed(); configFile != "" {
		h.localFilePath = configFile
	} else {
		h.localFilePath = "./config/dynconfig/example.yaml"
	}

	h.config = v
	h.readCurrentState()
	h.setupMelody()

	return h
}

func initKubeClient() clientset.Interface {
	rc, err := config.GetConfig()
	if err != nil {
		logger.Log.Info("[dynconfig] Running without cluster connection", "reason", err)
		return nil
	}
	client, err := clientset.NewForConfig(rc)
	if err != nil {
		logger.Log.Error(err, "[dynconfig] Failed to create Kubernetes client")
		return nil
	}
	return client
}

func (h *Handler) setupMelody() {
	h.melody.HandleConnect(func(s *melody.Session) {
		h.mu.RLock()
		msg := StateMessage{
			Type:          "init",
			RawContent:    h.rawContent,
			ConfigJSON:    h.formatConfigJSONLocked(),
			ConfigMapName: h.configMapName,
			Namespace:     h.namespace,
			InCluster:     h.kubeClient != nil,
			Events:        h.eventLog,
		}
		h.mu.RUnlock()

		if b, err := json.Marshal(msg); err == nil {
			_ = s.Write(b)
		}
	})
}

// Name implements handler.Handler.
func (h *Handler) Name() string {
	return "dynconfig"
}

// RegisterRoutes implements handler.Handler.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, ctx context.Context) {
	h.ctx, h.cancelCtx = context.WithCancel(ctx)
	mux.HandleFunc("/dynconfig", h.ServeHTTP)
	mux.HandleFunc("/dynconfig/apply", h.handleApply)
	mux.HandleFunc("/dynconfigws", func(w http.ResponseWriter, r *http.Request) {
		_ = h.melody.HandleRequest(w, r)
	})
	logger.Log.Info("[dynconfig] registered /dynconfig, /dynconfig/apply, /dynconfigws")

	go h.watchConfig()
}

// ServeHTTP handles the dynconfig UI request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	configJSON := h.formatConfigJSONLocked()
	raw := h.rawContent
	events := make([]FsEvent, len(h.eventLog))
	copy(events, h.eventLog)
	inCluster := h.kubeClient != nil
	cmName := h.configMapName
	ns := h.namespace
	h.mu.RUnlock()

	data := struct {
		assets.CommonData
		ConfigJSON    string
		RawContent    string
		ConfigMapName string
		Namespace     string
		InCluster     bool
		Events        []FsEvent
	}{
		CommonData:    assets.CommonData{Version: version.Version, Flag: version.Flag},
		ConfigJSON:    configJSON,
		RawContent:    raw,
		ConfigMapName: cmName,
		Namespace:     ns,
		InCluster:     inCluster,
		Events:        events,
	}

	if err := assets.ExecuteTemplate(w, "dynamicconfig.html", data); err != nil {
		logger.Log.Error(err, "[dynconfig] executing template")
	}
}

// handleApply processes changes from the UI editor and applies them to the cluster ConfigMap.
func (h *Handler) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	// Validate YAML syntax
	var parsed any
	if err := yaml.Unmarshal([]byte(req.Content), &parsed); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Invalid YAML: %v", err),
		})
		return
	}

	if h.kubeClient != nil {
		// Apply to Kubernetes ConfigMap
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		cm, err := h.kubeClient.CoreV1().ConfigMaps(h.namespace).Get(ctx, h.configMapName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			newCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      h.configMapName,
					Namespace: h.namespace,
					Labels: map[string]string{
						"app": "introspect",
					},
				},
				Data: map[string]string{
					"example.yaml": req.Content,
				},
			}
			_, err = h.kubeClient.CoreV1().ConfigMaps(h.namespace).Create(ctx, newCM, metav1.CreateOptions{})
		} else if err == nil {
			if cm.Data == nil {
				cm.Data = make(map[string]string)
			}
			cm.Data["example.yaml"] = req.Content
			_, err = h.kubeClient.CoreV1().ConfigMaps(h.namespace).Update(ctx, cm, metav1.UpdateOptions{})
		}

		if err != nil {
			logger.Log.Error(err, "[dynconfig] Failed to apply ConfigMap to cluster", "configMap", h.configMapName)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"error":   fmt.Sprintf("Kubernetes error: %v", err),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": fmt.Sprintf("ConfigMap '%s' updated in cluster (namespace: %s). Waiting for volume synchronization...", h.configMapName, h.namespace),
		})
		return
	}

	// Standalone/Local mode fallback: write to local file directly
	targetFile := h.localFilePath
	if targetFile == "" {
		targetFile = h.config.ConfigFileUsed()
	}
	if targetFile == "" {
		targetFile = "./config/dynconfig/example.yaml"
	}

	if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to create directory: %v", err),
		})
		return
	}

	if err := os.WriteFile(targetFile, []byte(req.Content), 0644); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to write file: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Local config saved to %s", targetFile),
	})
}

func (h *Handler) watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Log.Error(err, "[dynconfig] setup fsnotify watcher")
		return
	}
	h.watcher = watcher
	defer watcher.Close()

	configFile := h.config.ConfigFileUsed()
	if configFile != "" {
		_ = watcher.Add(configFile)
		dir := filepath.Dir(configFile)
		if dir != "" {
			_ = watcher.Add(dir)
		}
	} else {
		// Try watching default search dirs
		_ = watcher.Add("./config/dynconfig")
		_ = watcher.Add("/etc/config/dynconfig")
	}

	for {
		select {
		case e, ok := <-watcher.Events:
			if !ok {
				return
			}
			base := filepath.Base(e.Name)
			// Match example.yaml or k8s symlink swaps (..data, ..data_tmp, directory modifications)
			if base == "example.yaml" || base == "..data" || e.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0 {
				logger.Log.Info("[dynconfig] fsnotify event detected", "op", e.Op.String(), "file", e.Name)

				if e.Op&fsnotify.Remove == fsnotify.Remove || e.Op&fsnotify.Rename == fsnotify.Rename {
					_ = watcher.Remove(e.Name)
					time.Sleep(50 * time.Millisecond)
					_ = watcher.Add(e.Name)
				}

				time.Sleep(50 * time.Millisecond)
				h.handleFileChanged(e.Op.String(), filepath.Base(e.Name))
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Log.Error(err, "[dynconfig] Watcher error")
		case <-h.ctx.Done():
			return
		}
	}
}

func (h *Handler) handleFileChanged(op, file string) {
	h.mu.Lock()
	h.readCurrentState()

	evt := FsEvent{
		ID:        time.Now().UnixNano(),
		Timestamp: time.Now().Format("15:04:05"),
		Op:        op,
		File:      file,
		Message:   fmt.Sprintf("Event %s on %s", op, file),
	}
	h.eventLog = append(h.eventLog, evt)
	if len(h.eventLog) > 50 {
		h.eventLog = h.eventLog[len(h.eventLog)-50:]
	}

	msg := StateMessage{
		Type:          "fsnotify",
		RawContent:    h.rawContent,
		ConfigJSON:    h.formatConfigJSONLocked(),
		ConfigMapName: h.configMapName,
		Namespace:     h.namespace,
		InCluster:     h.kubeClient != nil,
		Event:         &evt,
	}
	h.mu.Unlock()

	if b, err := json.Marshal(msg); err == nil {
		_ = h.melody.Broadcast(b)
	}
}

func (h *Handler) readCurrentState() {
	configFile := h.config.ConfigFileUsed()
	if configFile != "" {
		if content, err := os.ReadFile(configFile); err == nil {
			h.rawContent = string(content)
		}
	}

	if err := h.config.ReadInConfig(); err != nil {
		logger.Log.Info("[dynconfig] reading config in current state", "error", err)
	}

	var parsed map[string]any
	if err := h.config.Unmarshal(&parsed); err == nil {
		h.example = parsed
	}
}

func (h *Handler) formatConfigJSONLocked() string {
	b, err := json.MarshalIndent(h.example, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// Close implements graceful shutdown.
func (h *Handler) Close() error {
	if h.cancelCtx != nil {
		h.cancelCtx()
	}
	if h.watcher != nil {
		_ = h.watcher.Close()
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
