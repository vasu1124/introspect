package guestbook

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/globalsign/mgo"
	"github.com/spf13/viper"
	"github.com/vasu1124/introspect/pkg/logger"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

// Handler .
type Handler struct {
	dbtype       string
	mgosession   *mgo.Collection
	etcdsession  *etcdv3.Client
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

	// don't ever close the mgosession. bad design. but good enough for demo
	// defer h.mgosession.Close()
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
				if h.mgosession == nil && h.etcdsession == nil {
					logger.Log.Info("[guestbook] session try connect ...")
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

	h.dbtype = config.Get("DBtype").(string)

	switch h.dbtype {
	case "mongodb":
		var dialInfo mgo.DialInfo
		err = config.Unmarshal(&dialInfo)
		if err != nil {
			logger.Log.Error(err, "[guestbook] unable to decode into Mongodb DialInfo struct")
			return
		}

		dialInfo.Username = string(username)
		dialInfo.Password = string(password)

		session, err := mgo.DialWithInfo(&dialInfo)
		if err != nil {
			logger.Log.Error(err, "[guestbook] Failed MongoDB", "Addrs", dialInfo.Addrs, "Database", dialInfo.Database)
			return
		} else {
			logger.Log.Info("[guestbook] Connected to MongoDB", "Addrs", dialInfo.Addrs, "Database", dialInfo.Database)
		}
		// Optional. Switch the session to a monotonic behavior.
		// session.SetMode(mgo.Monotonic, true)
		h.mgosession = session.DB("").C(dialInfo.Database)

	case "etcd":
		var etcdConfig etcdv3.Config
		err = config.Unmarshal(&etcdConfig)
		if err != nil {
			logger.Log.Error(err, "[guestbook] unable to decode into Etcdv3 Config struct")
			return
		}

		etcdConfig.Username = string(username)
		etcdConfig.Password = string(password)

		h.etcdsession, err = etcdv3.New(etcdConfig)
		if err != nil {
			logger.Log.Error(err, "[guestbook] Failed Etcdv3", "Endpoints", etcdConfig.Endpoints)
		} else {
			logger.Log.Info("[guestbook] Connected to Etcdv3", "Endpoints", etcdConfig.Endpoints)
		}

	}

}

// Entry .
type Entry struct {
	Date    time.Time
	Name    string
	Comment string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.mgosession == nil && h.etcdsession == nil {
		fmt.Fprint(w, "[guestbook] No Database Connection")
		logger.Log.Info("[guestbook] No Database Connection")
		return
	}

	t, err := template.ParseFiles("tmpl/guestbook.html")
	if err != nil {
		fmt.Fprint(w, "[guestbook] parsing template error")
		logger.Log.Error(err, "[guestbook] parsing template")
		return
	}

	err = r.ParseForm()
	if err != nil {
		fmt.Fprint(w, "[guestbook] parsing form error")
		logger.Log.Error(err, "[guestbook] parsing form")
	}

	var entries []Entry

	switch h.dbtype {
	case "mongodb":

		if r.Form["name"] != nil && r.Form["comment"] != nil &&
			r.Form["name"][0] != "" && r.Form["comment"][0] != "" {
			err = h.mgosession.Insert(&Entry{time.Now(), r.Form["name"][0], r.Form["comment"][0]})
			if err != nil {
				logger.Log.Error(err, "[guestbook] insert error")
			}
		}

		err = h.mgosession.Find(nil).All(&entries)
		if err != nil {
			logger.Log.Error(err, "[guestbook] find error")
		}

	case "etcd":
		if r.Form["name"] != nil && r.Form["comment"] != nil &&
			r.Form["name"][0] != "" && r.Form["comment"][0] != "" {
			entry := Entry{
				Date:    time.Now(),
				Name:    r.Form["name"][0],
				Comment: r.Form["comment"][0],
			}
			ret, err := json.Marshal(entry)
			if err != nil {
				logger.Log.Error(err, "[guestbook] marshall error")
				return
			}
			_, err = h.etcdsession.Put(context.Background(), entry.Name, string(ret))
			if err != nil {
				logger.Log.Error(err, "[guestbook] insert error")
			}
		}

		resp, err := h.etcdsession.Get(context.Background(), "", etcdv3.WithPrefix())
		if err != nil {
			logger.Log.Error(err, "[guestbook] get error")
		}

		for _, ev := range resp.Kvs {
			var entry Entry
			err = json.Unmarshal(ev.Value, &entry)
			if err != nil {
				logger.Log.Error(err, "[guestbook] unmarshall error")
			}
			entries = append(entries, entry)
		}
	}

	err = t.Execute(w, entries)
	if err != nil {
		logger.Log.Error(err, "[guestbook] executing template")
		fmt.Fprint(w, "[guestbook] executing template: ", err)
	}

}
