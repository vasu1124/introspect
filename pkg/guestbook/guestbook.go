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
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Handler .
type Handler struct {
	dbtype       string
	mongoClient  *mongo.Client
	mongoColl    *mongo.Collection
	etcdsession  *etcdv3.Client
	valkeyClient *redis.Client
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
				if h.mongoClient == nil && h.etcdsession == nil && h.valkeyClient == nil {
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
		addrs := config.GetStringSlice("Addrs")
		database := config.GetString("Database")
		if database == "" {
			database = "guestbook"
		}

		if len(addrs) == 0 {
			logger.Log.Error(nil, "[guestbook] No MongoDB addresses found in config")
			return
		}

		uri := fmt.Sprintf("mongodb://%s", strings.Join(addrs, ","))
		clientOpts := options.Client().ApplyURI(uri)
		clientOpts.SetAuth(options.Credential{
			Username: string(username),
			Password: string(password),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := mongo.Connect(ctx, clientOpts)
		if err != nil {
			logger.Log.Error(err, "[guestbook] Failed MongoDB Connect", "Addrs", addrs, "Database", database)
			h.mongoClient = nil
			h.mongoColl = nil
		} else {
			// Verify connection
			err = client.Ping(ctx, nil)
			if err != nil {
				logger.Log.Error(err, "[guestbook] Failed MongoDB Ping", "Addrs", addrs, "Database", database)
				h.mongoClient = nil
				h.mongoColl = nil
			} else {
				logger.Log.Info("[guestbook] Connected to MongoDB", "Addrs", addrs, "Database", database)
				h.mongoClient = client
				h.mongoColl = client.Database(database).Collection("guestbook")
			}
		}

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
			h.etcdsession = nil
		} else {
			logger.Log.Info("[guestbook] Connected to Etcdv3", "Endpoints", etcdConfig.Endpoints)
		}

	case "valkey":
		valkeyAddr := config.GetString("ValkeyAddr")
		if valkeyAddr == "" {
			valkeyAddr = "valkey:6379"
		}

		h.valkeyClient = redis.NewClient(&redis.Options{
			Addr:     valkeyAddr,
			Password: "", // no password set
			DB:       0,  // use default DB
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := h.valkeyClient.Ping(ctx).Err()
		if err != nil {
			logger.Log.Error(err, "[guestbook] Failed Valkey Ping", "Addr", valkeyAddr)
			h.valkeyClient = nil
		} else {
			logger.Log.Info("[guestbook] Connected to Valkey", "Addr", valkeyAddr)
		}
	}

}

// SwitchBackend switches the database backend between mongodb and etcd
func (h *Handler) SwitchBackend(backend string) error {
	if backend != "mongodb" && backend != "etcd" && backend != "valkey" {
		return fmt.Errorf("invalid backend: %s", backend)
	}

	// Close existing connections
	if h.mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.mongoClient.Disconnect(ctx)
		h.mongoClient = nil
		h.mongoColl = nil
	}
	if h.etcdsession != nil {
		h.etcdsession.Close()
		h.etcdsession = nil
	}
	if h.valkeyClient != nil {
		h.valkeyClient.Close()
		h.valkeyClient = nil
	}

	// Update the backend type
	h.dbtype = backend

	// Read config and reconnect
	config := viper.New()
	config.SetConfigName("config")
	config.AddConfigPath("/etc/config/")
	config.AddConfigPath("./etc/config")

	err := config.ReadInConfig()
	if err != nil {
		logger.Log.Error(err, "[guestbook] error reading config during switch")
		return err
	}

	// Temporarily override the DBtype in config
	config.Set("DBtype", backend)

	h.readConfig(config)

	logger.Log.Info("[guestbook] Switched backend", "backend", backend)
	return nil
}

// SwitchHandler handles the switch request
func (h *Handler) SwitchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		logger.Log.Error(err, "[guestbook] parsing form")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	backend := r.FormValue("backend")
	if backend == "" {
		http.Error(w, "Backend not specified", http.StatusBadRequest)
		return
	}

	err = h.SwitchBackend(backend)
	if err != nil {
		logger.Log.Error(err, "[guestbook] switching backend")
		http.Error(w, "Error switching backend", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/guestbook", http.StatusSeeOther)
}

// Entry .
type Entry struct {
	Date    time.Time
	Name    string
	Comment string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	t, err := template.ParseFiles("tmpl/layout.html", "tmpl/guestbook.html")
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
		if h.mongoColl != nil {
			if r.Form["name"] != nil && r.Form["comment"] != nil &&
				r.Form["name"][0] != "" && r.Form["comment"][0] != "" {
				_, err = h.mongoColl.InsertOne(context.Background(), Entry{time.Now(), r.Form["name"][0], r.Form["comment"][0]})
				if err != nil {
					logger.Log.Error(err, "[guestbook] insert error")
				}
			}

			cursor, err := h.mongoColl.Find(context.Background(), bson.D{})
			if err != nil {
				logger.Log.Error(err, "[guestbook] find error")
			} else {
				if err = cursor.All(context.Background(), &entries); err != nil {
					logger.Log.Error(err, "[guestbook] cursor all error")
				}
			}
		}

	case "etcd":
		if h.etcdsession != nil {
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
	case "valkey":
		if h.valkeyClient != nil {
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
				// Use a list to store entries
				err = h.valkeyClient.LPush(context.Background(), "guestbook", string(ret)).Err()
				if err != nil {
					logger.Log.Error(err, "[guestbook] valkey insert error")
				}
			}

			// Get all entries from the list
			valkeyEntries, err := h.valkeyClient.LRange(context.Background(), "guestbook", 0, -1).Result()
			if err != nil {
				logger.Log.Error(err, "[guestbook] valkey get error")
			}

			for _, ev := range valkeyEntries {
				var entry Entry
				err = json.Unmarshal([]byte(ev), &entry)
				if err != nil {
					logger.Log.Error(err, "[guestbook] unmarshall error")
				}
				entries = append(entries, entry)
			}
		}
	}

	// Create data structure for template
	type GuestbookData struct {
		Backend   string
		Connected bool
		Entries   []Entry
		Version   string
	}

	connected := (h.dbtype == "mongodb" && h.mongoClient != nil) || (h.dbtype == "etcd" && h.etcdsession != nil) || (h.dbtype == "valkey" && h.valkeyClient != nil)
	data := GuestbookData{
		Backend:   h.dbtype,
		Connected: connected,
		Entries:   entries,
		Version:   version.Get().GitVersion,
	}

	err = t.Execute(w, data)
	if err != nil {
		logger.Log.Error(err, "[guestbook] executing template")
		fmt.Fprint(w, "[guestbook] executing template: ", err)
	}

}
