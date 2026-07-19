package dynconfig

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/handler"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
)

// Handler implements handler.Handler for the dynamic config endpoint.
type Handler struct {
	example   map[string]string
	ctx       context.Context
	cancelCtx context.CancelFunc
	watcher   *fsnotify.Watcher
	config    *viper.Viper
}

// New creates a new dynconfig handler.
func New() *Handler {
	h := &Handler{}
	config := viper.New()

	config.SetConfigName("example")
	config.AddConfigPath("/etc/config/")
	config.AddConfigPath("./etc/config")
	if err := config.ReadInConfig(); err != nil {
		logger.Log.Error(err, "[dynconfig] Fatal error config file")
		return h
	}

	h.config = config
	h.readConfig(config)
	return h
}

// Name implements handler.Handler.
func (h *Handler) Name() string {
	return "dynconfig"
}

// RegisterRoutes implements handler.Handler.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, ctx context.Context) {
	h.ctx, h.cancelCtx = context.WithCancel(ctx)
	mux.HandleFunc("/dynconfig", h.ServeHTTP)
	logger.Log.Info("[dynconfig] registered /dynconfig")

	go h.watchConfig()
}

// ServeHTTP handles the dynconfig request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.example["OSENV_EXAMPLE"] = os.Getenv("OSENV_EXAMPLE")

	configJSON, err := json.MarshalIndent(h.example, "", "  ")
	if err != nil {
		logger.Log.Error(err, "[dynconfig] json marshal error")
		configJSON = []byte("{}")
	}

	data := struct {
		assets.CommonData
		ConfigJSON string
	}{
		CommonData: assets.CommonData{Version: version.Version, Flag: version.Flag},
		ConfigJSON: string(configJSON),
	}

	if err := assets.ExecuteTemplate(w, "dynamicconfig.html", data); err != nil {
		logger.Log.Error(err, "[dynconfig] executing template")
	}
}

func (h *Handler) watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Log.Error(err, "[dynconfig] setup fsnotify watcher")
		return
	}
	h.watcher = watcher
	defer watcher.Close()

	watcher.Add(h.config.ConfigFileUsed())

	for {
		select {
		case e := <-watcher.Events:
			if e.Op&fsnotify.Write == fsnotify.Write {
				logger.Log.Info("[dynconfig] config file changed", "file", e.Name)
				h.readConfig(h.config)
			}
			if e.Op&fsnotify.Remove == fsnotify.Remove {
				logger.Log.Info("[dynconfig] config file removed", "file", e.Name)
				watcher.Remove(e.Name)
				watcher.Add(e.Name)
				h.readConfig(h.config)
			}
		case err := <-watcher.Errors:
			logger.Log.Error(err, "[dynconfig] Watcher error")
		case <-h.ctx.Done():
			return
		}
	}
}

func (h *Handler) readConfig(config *viper.Viper) {
	if err := config.ReadInConfig(); err != nil {
		logger.Log.Error(err, "[dynconfig] error config file")
		return
	}
	if err := config.Unmarshal(&h.example); err != nil {
		logger.Log.Error(err, "[dynconfig] unable to unmarshal into struct")
	}
}

// Close implements graceful shutdown.
func (h *Handler) Close() error {
	if h.cancelCtx != nil {
		h.cancelCtx()
	}
	if h.watcher != nil {
		h.watcher.Close()
	}
	return nil
}

// Ensure Handler implements handler.Handler.
var _ handler.Handler = (*Handler)(nil)
