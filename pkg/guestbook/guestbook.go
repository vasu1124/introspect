package guestbook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/handler"
	"github.com/vasu1124/introspect/pkg/logger"
	"github.com/vasu1124/introspect/pkg/version"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Backend is the interface for guestbook storage backends.
type Backend interface {
	Insert(ctx context.Context, entry Entry) error
	Entries(ctx context.Context) ([]Entry, error)
	Close() error
	Ping(ctx context.Context) error
}

// Handler implements handler.Handler for the guestbook endpoint.
type Handler struct {
	mu        sync.RWMutex
	backend   Backend
	dbtype    string
	username  string
	password  string
	config    *viper.Viper
	watcher   *fsnotify.Watcher
	cancelCtx context.CancelFunc
}

// Entry represents a guestbook entry.
type Entry struct {
	Date    time.Time
	Name    string
	Comment string
}

// New creates a new guestbook handler.
func New() *Handler {
	h := &Handler{}

	config := viper.New()
	config.SetConfigName("config")
	config.AddConfigPath("/etc/config/")
	config.AddConfigPath("./etc/config")
	if err := config.ReadInConfig(); err != nil {
		logger.Log.Error(err, "[guestbook] Fatal error config file")
	}

	h.loadSecrets()
	h.config = config

	ctx, cancel := context.WithCancel(context.Background())
	h.cancelCtx = cancel

	// Initialize backend from config
	h.readConfig(config)

	// Start config watcher
	go h.watchConfig(ctx, config)

	return h
}

func (h *Handler) loadSecrets() {
	usernamefile, _ := filepath.Abs("etc/secret/username")
	if _, err := os.Stat(usernamefile); os.IsNotExist(err) {
		usernamefile, _ = filepath.Abs("/etc/secret/username")
	}
	passwordfile, _ := filepath.Abs("etc/secret/password")
	if _, err := os.Stat(passwordfile); os.IsNotExist(err) {
		passwordfile, _ = filepath.Abs("/etc/secret/password")
	}

	if u, err := os.ReadFile(usernamefile); err == nil {
		h.username = strings.TrimSpace(string(u))
	}
	if p, err := os.ReadFile(passwordfile); err == nil {
		h.password = strings.TrimSpace(string(p))
	}
}

func (h *Handler) readConfig(config *viper.Viper) {
	if err := config.ReadInConfig(); err != nil {
		logger.Log.Error(err, "[guestbook] fatal error config file")
		return
	}

	dbtype := config.GetString("DBtype")
	if dbtype == "" {
		dbtype = "mongodb"
	}

	username := h.username
	password := h.password

	var backend Backend
	var err error

	switch dbtype {
	case "mongodb":
		backend, err = newMongoBackend(config, username, password)
	case "etcd":
		backend, err = newEtcdBackend(config, username, password)
	case "valkey":
		backend, err = newValkeyBackend(config)
	default:
		logger.Log.Error(nil, "[guestbook] Unknown DB type", "type", dbtype)
	}

	if err != nil {
		logger.Log.Error(err, "[guestbook] Failed to init backend", "type", dbtype)
		return
	}

	h.mu.Lock()
	if h.backend != nil {
		h.backend.Close()
	}
	h.backend = backend
	h.dbtype = dbtype
	h.mu.Unlock()

	logger.Log.Info("[guestbook] Backend initialized", "type", dbtype)
}

func (h *Handler) watchConfig(ctx context.Context, config *viper.Viper) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Log.Error(err, "[guestbook] fsnotify error")
		return
	}
	h.watcher = watcher
	defer watcher.Close()

	watcher.Add(config.ConfigFileUsed())

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case e := <-watcher.Events:
			if e.Op&fsnotify.Write == fsnotify.Write {
				logger.Log.Info("[guestbook] Config file changed", "file", e.Name)
				h.readConfig(config)
			}
			if e.Op&fsnotify.Remove == fsnotify.Remove {
				logger.Log.Info("[guestbook] Config file removed", "file", e.Name)
				watcher.Remove(e.Name)
				watcher.Add(e.Name)
				h.readConfig(config)
			}
		case err := <-watcher.Errors:
			logger.Log.Error(err, "[guestbook] Watcher error")
		case <-ticker.C:
			h.mu.RLock()
			needsReconnect := h.backend == nil
			h.mu.RUnlock()
			if needsReconnect {
				h.readConfig(config)
			}
		case <-ctx.Done():
			return
		}
	}
}

// Name implements handler.Handler.
func (h *Handler) Name() string {
	return "guestbook"
}

// RegisterRoutes implements handler.Handler.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, ctx context.Context) {
	mux.HandleFunc("/guestbook", h.ServeHTTP)
	mux.HandleFunc("/guestbook/switch", h.SwitchHandler)
	logger.Log.Info("[guestbook] registered /guestbook and /guestbook/switch")
}

// ServeHTTP handles the guestbook request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	backend := h.backend
	dbtype := h.dbtype
	h.mu.RUnlock()

	var entries []Entry

	if backend != nil {
		if r.Form == nil {
			r.ParseForm()
		}

		if r.Form["name"] != nil && r.Form["comment"] != nil &&
			r.Form["name"][0] != "" && r.Form["comment"][0] != "" {
			entry := Entry{
				Date:    time.Now(),
				Name:    r.Form["name"][0],
				Comment: r.Form["comment"][0],
			}
			if err := backend.Insert(r.Context(), entry); err != nil {
				logger.Log.Error(err, "[guestbook] insert error")
			}
		}

		entries, _ = backend.Entries(r.Context())
	}

	connected := backend != nil

	data := struct {
		assets.CommonData
		Backend   string
		Connected bool
		Entries   []Entry
	}{
		CommonData: assets.CommonData{Version: version.Version, Flag: version.Flag},
		Backend:    dbtype,
		Connected:  connected,
		Entries:    entries,
	}

	if err := assets.ExecuteTemplate(w, "guestbook.html", data); err != nil {
		logger.Log.Error(err, "[guestbook] executing template")
	}
}

// SwitchHandler handles backend switching.
func (h *Handler) SwitchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		logger.Log.Error(err, "[guestbook] parsing form")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	backend := r.FormValue("backend")
	if backend == "" {
		http.Error(w, "Backend not specified", http.StatusBadRequest)
		return
	}

	if err := h.SwitchBackend(backend); err != nil {
		logger.Log.Error(err, "[guestbook] switching backend")
		http.Error(w, "Error switching backend", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/guestbook", http.StatusSeeOther)
}

// SwitchBackend switches the database backend.
func (h *Handler) SwitchBackend(backendType string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if backendType != "mongodb" && backendType != "etcd" && backendType != "valkey" {
		return fmt.Errorf("invalid backend: %s", backendType)
	}

	if h.backend != nil {
		if err := h.backend.Close(); err != nil {
			logger.Log.Error(err, "[guestbook] closing old backend")
		}
	}

	var backend Backend
	var err error

	switch backendType {
	case "mongodb":
		backend, err = newMongoBackend(h.config, h.username, h.password)
	case "etcd":
		backend, err = newEtcdBackend(h.config, h.username, h.password)
	case "valkey":
		backend, err = newValkeyBackend(h.config)
	}

	h.backend = backend
	h.dbtype = backendType

	if err != nil {
		logger.Log.Error(err, "[guestbook] Failed to init backend", "type", backendType)
		return err
	}

	logger.Log.Info("[guestbook] Switched backend", "backend", backendType)
	return nil
}

// Close implements graceful shutdown.
func (h *Handler) Close() error {
	if h.cancelCtx != nil {
		h.cancelCtx()
	}
	if h.watcher != nil {
		h.watcher.Close()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.backend != nil {
		return h.backend.Close()
	}
	return nil
}

// Ensure Handler implements handler.Handler.
var _ handler.Handler = (*Handler)(nil)

// --- MongoDB Backend ---

type mongoBackend struct {
	client *mongo.Client
	coll   *mongo.Collection
}

func newMongoBackend(config *viper.Viper, username, password string) (*mongoBackend, error) {
	addrs := config.GetStringSlice("Addrs")
	database := config.GetString("Database")
	if database == "" {
		database = "guestbook"
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("no MongoDB addresses in config")
	}

	uri := fmt.Sprintf("mongodb://%s", strings.Join(addrs, ","))
	clientOpts := options.Client().ApplyURI(uri)
	clientOpts.SetAuth(options.Credential{
		Username: username,
		Password: password,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	coll := client.Database(database).Collection("guestbook")
	logger.Log.Info("[guestbook] Connected to MongoDB", "Addrs", addrs, "Database", database)
	return &mongoBackend{client: client, coll: coll}, nil
}

func (b *mongoBackend) Insert(ctx context.Context, entry Entry) error {
	_, err := b.coll.InsertOne(ctx, entry)
	return err
}

func (b *mongoBackend) Entries(ctx context.Context) ([]Entry, error) {
	cursor, err := b.coll.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var entries []Entry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (b *mongoBackend) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return b.client.Disconnect(ctx)
}

func (b *mongoBackend) Ping(ctx context.Context) error {
	return b.client.Ping(ctx, nil)
}

// --- etcd Backend ---

type etcdBackend struct {
	client *etcdv3.Client
}

func newEtcdBackend(config *viper.Viper, username, password string) (*etcdBackend, error) {
	var etcdConfig etcdv3.Config
	if err := config.Unmarshal(&etcdConfig); err != nil {
		return nil, err
	}

	etcdConfig.Username = username
	etcdConfig.Password = password

	client, err := etcdv3.New(etcdConfig)
	if err != nil {
		return nil, err
	}

	logger.Log.Info("[guestbook] Connected to Etcdv3", "Endpoints", etcdConfig.Endpoints)
	return &etcdBackend{client: client}, nil
}

func (b *etcdBackend) Insert(ctx context.Context, entry Entry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = b.client.Put(ctx, entry.Name, string(data))
	return err
}

func (b *etcdBackend) Entries(ctx context.Context) ([]Entry, error) {
	resp, err := b.client.Get(ctx, "", etcdv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, kv := range resp.Kvs {
		var entry Entry
		if err := json.Unmarshal(kv.Value, &entry); err != nil {
			logger.Log.Error(err, "[guestbook] etcd unmarshall error")
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (b *etcdBackend) Close() error {
	return b.client.Close()
}

func (b *etcdBackend) Ping(ctx context.Context) error {
	_, err := b.client.Status(ctx, b.client.Endpoints()[0])
	return err
}

// --- Valkey/Redis Backend ---

type valkeyBackend struct {
	client *redis.Client
}

func newValkeyBackend(config *viper.Viper) (*valkeyBackend, error) {
	addr := config.GetString("ValkeyAddr")
	if addr == "" {
		addr = "valkey:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	logger.Log.Info("[guestbook] Connected to Valkey", "Addr", addr)
	return &valkeyBackend{client: client}, nil
}

func (b *valkeyBackend) Insert(ctx context.Context, entry Entry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return b.client.LPush(ctx, "guestbook", string(data)).Err()
}

func (b *valkeyBackend) Entries(ctx context.Context) ([]Entry, error) {
	vals, err := b.client.LRange(ctx, "guestbook", 0, -1).Result()
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, v := range vals {
		var entry Entry
		if err := json.Unmarshal([]byte(v), &entry); err != nil {
			logger.Log.Error(err, "[guestbook] valkey unmarshall error")
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (b *valkeyBackend) Close() error {
	return b.client.Close()
}

func (b *valkeyBackend) Ping(ctx context.Context) error {
	return b.client.Ping(ctx).Err()
}
