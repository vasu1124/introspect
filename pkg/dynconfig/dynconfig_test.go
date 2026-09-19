package dynconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/vasu1124/introspect/pkg/assets"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMain(m *testing.M) {
	if _, err := os.Stat("../../tmpl"); err == nil {
		assets.TemplateDir = "../../tmpl"
	}
	os.Exit(m.Run())
}

func TestDynconfigHandlerBasics(t *testing.T) {
	h := NewWithClient(nil)
	defer h.Close()

	if h.Name() != "dynconfig" {
		t.Fatalf("expected name 'dynconfig', got %s", h.Name())
	}

	mux := http.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h.RegisterRoutes(mux, ctx)

	// Test GET /dynconfig
	req := httptest.NewRequest(http.MethodGet, "/dynconfig", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Dynamic Configuration") {
		t.Errorf("expected body to contain 'Dynamic Configuration'")
	}
	if !strings.Contains(body, "introspect-dynconfig") {
		t.Errorf("expected body to contain 'introspect-dynconfig'")
	}
	// Verify OSENV_EXAMPLE is completely removed
	if strings.Contains(body, "OSENV_EXAMPLE") {
		t.Errorf("body should NOT contain 'OSENV_EXAMPLE'")
	}
}

func TestDynconfigApplyValidation(t *testing.T) {
	h := NewWithClient(nil)
	defer h.Close()

	mux := http.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.RegisterRoutes(mux, ctx)

	// Test invalid method GET on /dynconfig/apply
	req := httptest.NewRequest(http.MethodGet, "/dynconfig/apply", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got %d", rr.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/dynconfig/apply", strings.NewReader("bad json"))
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}

	// Test invalid YAML
	reqBody, _ := json.Marshal(map[string]string{
		"content": "key: [unclosed bracket",
	})
	req = httptest.NewRequest(http.MethodPost, "/dynconfig/apply", bytes.NewReader(reqBody))
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for invalid YAML, got %d", rr.Code)
	}
}

func TestDynconfigApplyWithKubeClient(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	h := NewWithClient(fakeClient)
	defer h.Close()

	h.namespace = "test-ns"
	h.configMapName = "introspect-dynconfig"

	mux := http.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.RegisterRoutes(mux, ctx)

	// Apply valid YAML when ConfigMap doesn't exist yet -> should create
	yamlContent := "foo: bar\ntimeout: 25\n"
	reqBody, _ := json.Marshal(map[string]string{
		"content": yamlContent,
	})
	req := httptest.NewRequest(http.MethodPost, "/dynconfig/apply", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Verify ConfigMap was created in fakeClient
	cm, err := fakeClient.CoreV1().ConfigMaps("test-ns").Get(context.Background(), "introspect-dynconfig", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get created ConfigMap: %v", err)
	}
	if cm.Data["example.yaml"] != yamlContent {
		t.Errorf("expected ConfigMap data to be %q, got %q", yamlContent, cm.Data["example.yaml"])
	}

	// Update existing ConfigMap
	updatedYAML := "foo: baz\ntimeout: 99\n"
	reqBody2, _ := json.Marshal(map[string]string{
		"content": updatedYAML,
	})
	req2 := httptest.NewRequest(http.MethodPost, "/dynconfig/apply", bytes.NewReader(reqBody2))
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr2.Code)
	}

	cm2, err := fakeClient.CoreV1().ConfigMaps("test-ns").Get(context.Background(), "introspect-dynconfig", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get updated ConfigMap: %v", err)
	}
	if cm2.Data["example.yaml"] != updatedYAML {
		t.Errorf("expected ConfigMap data to be %q, got %q", updatedYAML, cm2.Data["example.yaml"])
	}
}

func TestDynconfigApplyStandaloneFile(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "example.yaml")

	h := NewWithClient(nil)
	defer h.Close()

	h.kubeClient = nil
	h.localFilePath = tempFile

	mux := http.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.RegisterRoutes(mux, ctx)

	yamlContent := "setting: enabled\ncount: 42\n"
	reqBody, _ := json.Marshal(map[string]string{
		"content": yamlContent,
	})
	req := httptest.NewRequest(http.MethodPost, "/dynconfig/apply", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	readBack, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(readBack) != yamlContent {
		t.Errorf("expected file content %q, got %q", yamlContent, string(readBack))
	}
}

func TestDynconfigConcurrentSafety(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "introspect-dynconfig",
			Namespace: "default",
		},
		Data: map[string]string{
			"example.yaml": "initial: true\n",
		},
	})

	h := NewWithClient(fakeClient)
	defer h.Close()

	var wg sync.WaitGroup
	// Concurrently read state and trigger simulated change events
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/dynconfig", nil)
			h.ServeHTTP(rr, req)

			h.handleFileChanged("WRITE", "example.yaml")
		}(i)
	}
	wg.Wait()
}
