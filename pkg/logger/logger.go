package logger

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/vasu1124/introspect/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
)

// Log is the main logger
var Log logr.Logger

// LogEntry represents a captured log entry from zap
type LogEntry struct {
	Time    time.Time
	Level   string
	Message string
	Fields  map[string]any
}

// LogSubscriber is a callback function for receiving log entries
type LogSubscriber func(entry LogEntry)

var (
	subscribersMu sync.RWMutex
	subscribers   = make(map[uint64]LogSubscriber)
	nextSubID     uint64
)

// Subscribe registers a subscriber that receives log entries.
// It returns an unsubscribe function.
func Subscribe(sub LogSubscriber) func() {
	subscribersMu.Lock()
	defer subscribersMu.Unlock()
	nextSubID++
	id := nextSubID
	subscribers[id] = sub
	return func() {
		subscribersMu.Lock()
		delete(subscribers, id)
		subscribersMu.Unlock()
	}
}

type broadcastCore struct {
	zapcore.LevelEnabler
}

func (c *broadcastCore) With(fields []zapcore.Field) zapcore.Core {
	return c
}

func (c *broadcastCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

func (c *broadcastCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	subscribersMu.RLock()
	if len(subscribers) == 0 {
		subscribersMu.RUnlock()
		return nil
	}
	subs := make([]LogSubscriber, 0, len(subscribers))
	for _, s := range subscribers {
		subs = append(subs, s)
	}
	subscribersMu.RUnlock()

	fieldMap := make(map[string]any, len(fields))
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(enc)
	}
	for k, v := range enc.Fields {
		fieldMap[k] = v
	}

	le := LogEntry{
		Time:    entry.Time,
		Level:   entry.Level.String(),
		Message: entry.Message,
		Fields:  fieldMap,
	}

	for _, s := range subs {
		s(le)
	}
	return nil
}

func (c *broadcastCore) Sync() error {
	return nil
}

func init() {
	InitZap()
}

// InitZap initializes the Zap based logging framework
func InitZap() {
	zapConfig := zap.NewProductionConfig()
	if config.Default.Development {
		zapConfig = zap.NewDevelopmentConfig()
	}

	if config.Default.LogLevel == "" {
		config.Default.LogLevel = zapConfig.Level.String()
	}

	level := zapConfig.Level
	switch config.Default.LogLevel {
	case "debug":
		level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "fatal":
		level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	case "panic":
		level = zap.NewAtomicLevelAt(zapcore.PanicLevel)
	default:
		fmt.Fprintf(os.Stderr, "log-level not recocnized\n")
	}

	zapConfig.Level = level

	bCore := &broadcastCore{LevelEnabler: level}
	log, _ := zapConfig.Build(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(c, bCore)
	}))
	Log = zapr.NewLogger(log)

	klog.SetLogger(Log)
}

