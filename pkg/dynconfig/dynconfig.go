package dynconfig

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
)

// Handler .
type Handler struct {
	example map[string]string
}

// New .
func New() *Handler {
	var h Handler
	config := viper.New()

	config.SetConfigName("example")      // name of config file (without extension)
	config.AddConfigPath("/etc/config/") // path to look for the config file in
	config.AddConfigPath("./etc/config") // call multiple times to add many search paths
	err := config.ReadInConfig()         // Find and read the config file
	if err != nil {                      // Handle errors reading the config file
		logger.Log.Error(err, "[dynconfig] Fatal error config file")
		return &h
	}

	//initally read and configure
	h.readConfig(config)

	//Establish a watch on config files
	go h.watchConfig(config)

	return &h
}

func (h *Handler) watchConfig(config *viper.Viper) {
	// Do not use config.WatchConfig()... in pod:
	// The config file that ConfigMap mounts in the pod is actually a symlink to a version of our config file.
	// Thus when ConfigMap updates occur, kubernetes' AtomicWriter() can achieve atomic ConfigMap updates as follows:
	// AtomicWriter() creates a new directory. Writes the updated ConfigMap to the new directory.
	// Once the write is complete it removes the original config file symlink.
	// And replaces it with a new symlink pointing to the contents of the newly created directory.	config.WatchConfig()

	// config.WatchConfig()
	// config.OnConfigChange(func(e fsnotify.Event) {
	// 	log.Println("Config file changed:", e.Name)
	// 	h.readConfig(config)
	// })

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Log.Error(err, "[dynconfig] setup fsnotify watcher")
	}
	defer watcher.Close()

	watcher.Add(config.ConfigFileUsed())

	done := make(chan bool)
	go func() {
		for {
			select {
			case e := <-watcher.Events:
				if e.Op&fsnotify.Write == fsnotify.Write { //standard change
					logger.Log.Info("[dynconfig] config file changed", "file", e.Name)
					h.readConfig(config)
				}
				if e.Op&fsnotify.Remove == fsnotify.Remove { //symlink remove change
					logger.Log.Info("[dynconfig] config file removed", "file", e.Name)
					watcher.Remove(e.Name)
					watcher.Add(e.Name)
					h.readConfig(config)
				}
			case err := <-watcher.Errors:
				logger.Log.Error(err, "[dynconfig] Watcher error")
			}
		}
	}()

	<-done
}

func (h *Handler) readConfig(config *viper.Viper) {
	err := config.ReadInConfig()
	if err != nil {
		logger.Log.Error(err, "[dynconfig] error config file")
		return
	}
	err = config.Unmarshal(&h.example)
	if err != nil {
		logger.Log.Error(err, "[dynconfig] unable to unmarshal into struct")
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.example["OSENV_EXAMPLE"] = os.Getenv("OSENV_EXAMPLE")

	t, err := template.ParseFiles("tmpl/layout.html", "tmpl/dynamicconfig.html")
	if err != nil {
		logger.Log.Error(err, "[dynconfig] template parse error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	configJSON, err := json.MarshalIndent(h.example, "", "  ")
	if err != nil {
		logger.Log.Error(err, "[dynconfig] json marshal error")
		configJSON = []byte("{}")
	}

	type PageData struct {
		Version    string
		Flag       bool
		ConfigJSON string
	}

	data := PageData{
		Version:    version.Get().GitVersion,
		Flag:       version.GetPatchVersion()%2 == 0,
		ConfigJSON: string(configJSON),
	}

	err = t.Execute(w, data)
	if err != nil {
		logger.Log.Error(err, "[dynconfig] executing template")
		fmt.Fprint(w, "[dynconfig] executing template: ", err)
	}
}
