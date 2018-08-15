package dynconfig

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
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
		log.Printf("[dynconfig] Fatal error config file: %s \n", err)
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
		log.Fatal("[dynconfig] Fatal: ", err)
	}
	defer watcher.Close()

	watcher.Add(config.ConfigFileUsed())

	done := make(chan bool)
	go func() {
		for {
			select {
			case e := <-watcher.Events:
				if e.Op&fsnotify.Write == fsnotify.Write { //standard change
					log.Println("[dynconfig] Config file changed: ", e.Name)
					h.readConfig(config)
				}
				if e.Op&fsnotify.Remove == fsnotify.Remove { //symlink remove change
					log.Println("[dynconfig] Config file removed: ", e.Name)
					watcher.Remove(e.Name)
					watcher.Add(e.Name)
					h.readConfig(config)
				}
			case err := <-watcher.Errors:
				log.Println("[dynconfig] Watcher error: ", err)
			}
		}
	}()

	<-done
}

func (h *Handler) readConfig(config *viper.Viper) {
	err := config.ReadInConfig()
	if err != nil {
		log.Printf("[dynconfig] Fatal error config file: %v", err)
		return
	}
	err = config.Unmarshal(&h.example)
	if err != nil {
		log.Printf("[dynconfig] unable to decode into struct: %v", err)
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.example["OSENV_EXAMPLE"] = os.Getenv("OSENV_EXAMPLE")
	x, _ := json.Marshal(h.example)
	fmt.Fprintf(w, "%s", x)
}
