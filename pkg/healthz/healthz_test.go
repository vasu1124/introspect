package healthz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	gorilla "github.com/gorilla/websocket"
	"github.com/vasu1124/introspect/pkg/assets"
	"github.com/vasu1124/introspect/pkg/logger"
)

func TestMain(m *testing.M) {
	if _, err := os.Stat("../../tmpl"); err == nil {
		assets.TemplateDir = "../../tmpl"
	}
	os.Exit(m.Run())
}

func TestHealthzHandlerBasics(t *testing.T) {
	h := New()
	defer h.Close()

	if h.Name() != "healthz" {
		t.Fatalf("expected name 'healthz', got %s", h.Name())
	}

	mux := http.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h.RegisterRoutes(mux, ctx)

	// Test GET /healthz
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Status: 200") {
		t.Errorf("expected body to contain 'Status: 200', got %q", rr.Body.String())
	}

	// Test GET /healthzr
	req = httptest.NewRequest(http.MethodGet, "/healthzr", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Status: 200") {
		t.Errorf("expected body to contain 'Status: 200', got %q", rr.Body.String())
	}

	// Test GET /healthz/ui
	req = httptest.NewRequest(http.MethodGet, "/healthz/ui", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /healthz/ui, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Health Checks") {
		t.Errorf("expected body to contain 'Health Checks'")
	}
}

func TestHealthzQueryParamToggles(t *testing.T) {
	h := New()
	defer h.Close()

	mux := http.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.RegisterRoutes(mux, ctx)

	// 1. Die liveness
	req := httptest.NewRequest(http.MethodGet, "/healthz?die", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for /healthz?die, got %d", rr.Code)
	}
	if h.getLivenessStatus() != http.StatusInternalServerError {
		t.Fatalf("expected liveness status 500, got %d", h.getLivenessStatus())
	}

	// Subsequent check without params should also return 500
	req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for /healthz, got %d", rr.Code)
	}

	// 2. Revive liveness
	req = httptest.NewRequest(http.MethodGet, "/healthz?live", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /healthz?live, got %d", rr.Code)
	}
	if h.getLivenessStatus() != http.StatusOK {
		t.Fatalf("expected liveness status 200, got %d", h.getLivenessStatus())
	}

	// 3. Die readiness
	req = httptest.NewRequest(http.MethodGet, "/healthzr?die", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for /healthzr?die, got %d", rr.Code)
	}
	if h.getReadinessStatus() != http.StatusServiceUnavailable {
		t.Fatalf("expected readiness status 503, got %d", h.getReadinessStatus())
	}

	// 4. Revive readiness
	req = httptest.NewRequest(http.MethodGet, "/healthzr?live", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /healthzr?live, got %d", rr.Code)
	}
	if h.getReadinessStatus() != http.StatusOK {
		t.Fatalf("expected readiness status 200, got %d", h.getReadinessStatus())
	}
}

func TestHealthzProbeLoggingAndFiltering(t *testing.T) {
	h := New()
	defer h.Close()

	mux := http.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.RegisterRoutes(mux, ctx)

	// Send a request with a custom user agent (simulating kube-probe)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("User-Agent", "kube-probe/1.28")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	logs := h.getRecentLogs()
	if len(logs) == 0 {
		t.Fatalf("expected at least 1 probe log, got %d", len(logs))
	}

	last := logs[len(logs)-1]
	if last.Type != "liveness" {
		t.Errorf("expected type 'liveness', got %s", last.Type)
	}
	if last.UserAgent != "kube-probe/1.28" {
		t.Errorf("expected UserAgent 'kube-probe/1.28', got %s", last.UserAgent)
	}
	if last.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", last.Status)
	}

	// Emit a non-probe log through logger.Log
	initialCount := len(h.getRecentLogs())
	logger.Log.Info("[metrics] scraped /metrics successfully", "path", "/metrics")

	// Small pause to allow callback processing
	time.Sleep(20 * time.Millisecond)
	if len(h.getRecentLogs()) != initialCount {
		t.Errorf("non-probe log should have been filtered out, count changed from %d to %d", initialCount, len(h.getRecentLogs()))
	}

	// Emit a log tagged with [healthz] that is not a duplicate probe log
	logger.Log.Info("[healthz] external probe test log alert")
	time.Sleep(20 * time.Millisecond)
	newLogs := h.getRecentLogs()
	if len(newLogs) <= initialCount {
		t.Errorf("expected probe-filtered log to be captured, count remained %d", len(newLogs))
	}
	captured := newLogs[len(newLogs)-1]
	if !strings.Contains(captured.Message, "external probe test log alert") {
		t.Errorf("expected captured message to contain 'external probe test log alert', got %q", captured.Message)
	}
}

func TestHealthzWebSocketFeed(t *testing.T) {
	h := New()
	defer h.Close()

	mux := http.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.RegisterRoutes(mux, ctx)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/healthzws"

	conn, _, err := gorilla.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket at %s: %v", wsURL, err)
	}
	defer conn.Close()

	// 1. Read initial "init" message
	var initMsg WebSocketMessage
	err = conn.ReadJSON(&initMsg)
	if err != nil {
		t.Fatalf("failed to read init message: %v", err)
	}
	if initMsg.Type != "init" {
		t.Errorf("expected init message type, got %s", initMsg.Type)
	}

	// 2. Trigger a probe request to generate a live log
	client := server.Client()
	resp, err := client.Get(server.URL + "/healthz?die")
	if err != nil {
		t.Fatalf("failed to GET /healthz?die: %v", err)
	}
	resp.Body.Close()

	// 3. Read live log message over websocket
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var liveMsg WebSocketMessage
	err = conn.ReadJSON(&liveMsg)
	if err != nil {
		t.Fatalf("failed to read live message over websocket: %v", err)
	}

	if liveMsg.Type != "log" {
		t.Errorf("expected type 'log', got %s", liveMsg.Type)
	}
	if liveMsg.Log == nil {
		t.Fatalf("expected Log field to be non-nil")
	}
	if liveMsg.Log.Status != http.StatusInternalServerError {
		t.Errorf("expected log status 500, got %d", liveMsg.Log.Status)
	}
	if liveMsg.LivenessStatus != http.StatusInternalServerError {
		t.Errorf("expected livenessStatus 500, got %d", liveMsg.LivenessStatus)
	}
}

func TestHealthzConcurrentSafety(t *testing.T) {
	h := New()
	defer h.Close()

	mux := http.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.RegisterRoutes(mux, ctx)

	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(3)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
		}(i)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/healthzr", nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)
		}(i)
		go func(idx int) {
			defer wg.Done()
			_ = h.getRecentLogs()
			_ = h.getLivenessStatus()
			_ = h.getReadinessStatus()
		}(i)
	}
	wg.Wait()
}
