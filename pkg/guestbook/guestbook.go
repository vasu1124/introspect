package guestbook

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/vasu1124/introspect/pkg/logger"
	"gopkg.in/mgo.v2"
)

// Handler .
type Handler struct {
	session      *mgo.Session
	database     string
	usernamefile string
	passwordfile string
}

// New .
func New() *Handler {
	var h Handler
	config := viper.New()

	config.SetConfigName("config")       // name of config file (without extension)
	config.AddConfigPath("/etc/config/") // path to look for the config file in
	config.AddConfigPath("./etc/config") // call multiple times to add many search paths
	err := config.ReadInConfig()         // Find and read the config file
	if err != nil {                      // Handle errors reading the config file
		logger.Log.Error(err, "[guestbook] Fatal error config file")
	}

	h.usernamefile, _ = filepath.Abs("etc/secret/username") //try relative
	if _, err := os.Stat(h.usernamefile); os.IsNotExist(err) {
		h.usernamefile, _ = filepath.Abs("/etc/secret/username") //try absolute
	}

	h.passwordfile, _ = filepath.Abs("etc/secret/password") //try relative
	if _, err := os.Stat(h.passwordfile); os.IsNotExist(err) {
		h.passwordfile, _ = filepath.Abs("/etc/secret/password") //try absolute
	}

	//initally read and configure
	h.readConfig(config)

	//Establish a watch on config, username & password files
	go h.watchConfig(config)

	// don't ever close the session. bad design. but good enough for demo
	// defer h.session.Close()
	return &h
}

func (h *Handler) watchConfig(config *viper.Viper) {
	// Do not use config.WatchConfig()... in pod:
	// The config file that ConfigMap mounts in the pod is actually a symlink to a version of our config file.
	// Thus when ConfigMap updates occur, kubernetes' AtomicWriter() can achieve atomic ConfigMap updates as follows:
	// AtomicWriter() creates a new directory. Writes the updated ConfigMap to the new directory.
	// Once the write is complete it removes the original config file symlink.
	// And replaces it with a new symlink pointing to the contents of the newly created directory.

	// config.WatchConfig()
	// config.OnConfigChange(func(e fsnotify.Event) {
	// 	log.Println("Config file changed:", e.Name)
	// 	h.readConfig(config)
	// })

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Log.Error(err, "[guestbook] fsnotify error")
	}
	defer watcher.Close()

	watcher.Add(h.usernamefile)
	watcher.Add(h.passwordfile)
	watcher.Add(config.ConfigFileUsed())

	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case e := <-watcher.Events:
				if e.Op&fsnotify.Write == fsnotify.Write { //standard change
					logger.Log.Info("[guestbook] Config file changed", "file", e.Name)
					h.readConfig(config)
				}
				if e.Op&fsnotify.Remove == fsnotify.Remove { //symlink remove change
					logger.Log.Info("[guestbook] Config file removed", "file", e.Name)
					watcher.Remove(e.Name)
					watcher.Add(e.Name)
					h.readConfig(config)
				}
			case err := <-watcher.Errors:
				logger.Log.Error(err, "[guestbook] Watcher error")
			case <-ticker.C:
				if h.session == nil {
					logger.Log.Info("[guestbook] MongoDB session try connect ...")
					h.readConfig(config)
				}
			}
		}
	}()

	<-done
}

func (h *Handler) readConfig(config *viper.Viper) {
	err := config.ReadInConfig()
	if err != nil {
		logger.Log.Error(err, "[guestbook] fatal error config file")
		return
	}
	var dialInfo mgo.DialInfo
	err = config.Unmarshal(&dialInfo)
	if err != nil {
		logger.Log.Error(err, "[guestbook] unable to decode into struct")
	}

	username, err := ioutil.ReadFile(h.usernamefile)
	if err != nil {
		logger.Log.Error(err, "[guestbook] file error")
		return
	}

	password, err := ioutil.ReadFile(h.passwordfile)
	if err != nil {
		logger.Log.Error(err, "[guestbook] file error")
		return
	}

	dialInfo.Username = string(username)
	dialInfo.Password = string(password)

	h.session, err = mgo.DialWithInfo(&dialInfo)
	if err != nil {
		logger.Log.Info("[guestbook] MongoDB", "dialInfo", dialInfo)
	} else {
		logger.Log.Info("[guestbook] Connected to MongoDB dialInfo", "dialInfo", dialInfo)
	}

	// Optional. Switch the session to a monotonic behavior.
	// h.session.SetMode(mgo.Monotonic, true)
}

// Entry .
type Entry struct {
	Date    time.Time
	Name    string
	Comment string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.session == nil {
		fmt.Fprint(w, "[guestbook] No Mongo Database Connection")
		return
	}

	t, err := template.ParseFiles("tmpl/guestbook.html")
	if err != nil {
		logger.Log.Error(err, "[guestbook] parsing template")
		return
	}

	err = r.ParseForm()
	if err != nil {
		logger.Log.Error(err, "[guestbook] parsing form")
	}

	c := h.session.DB(h.database).C("guestbook")

	if r.Form["name"] != nil && r.Form["comment"] != nil &&
		r.Form["name"][0] != "" && r.Form["comment"][0] != "" {
		err = c.Insert(&Entry{time.Now(), r.Form["name"][0], r.Form["comment"][0]})
		if err != nil {
			logger.Log.Error(err, "[guestbook] insert error")
		}
	}

	var entries []Entry
	err = c.Find(nil).All(&entries)
	if err != nil {
		logger.Log.Error(err, "[guestbook] find error")
	}

	err = t.Execute(w, entries)
	if err != nil {
		logger.Log.Error(err, "[guestbook] executing template")
		fmt.Fprint(w, "[guestbook] executing template: ", err)
	}

}
