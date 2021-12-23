package logger

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/vasu1124/introspect/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
)

var Log logr.Logger

func Printf(format string, args ...interface{}) {
	Log.Info(fmt.Sprintf(format, args...))
}

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

	log, _ := zapConfig.Build()
	Log = zapr.NewLogger(log)

	klog.SetLogger(Log)
}
