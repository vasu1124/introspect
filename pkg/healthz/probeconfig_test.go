package healthz

import (
	"strings"
	"testing"
)

func TestDefaultProbeConfigMatchesDeployment(t *testing.T) {
	pc := defaultProbeConfig()
	if pc.Source != "default" {
		t.Errorf("expected source 'default', got %s", pc.Source)
	}
	// Mirror kubernetes/introspect/deployment.yaml
	wantL := ProbeSpec{
		Path: "/healthz", Port: 9090, Scheme: "HTTPS",
		InitialDelaySeconds: 15, PeriodSeconds: 20, TimeoutSeconds: 2,
		SuccessThreshold: 1, FailureThreshold: 3,
	}
	wantR := ProbeSpec{
		Path: "/healthzr", Port: 9090, Scheme: "HTTPS",
		InitialDelaySeconds: 16, PeriodSeconds: 15, TimeoutSeconds: 3,
		SuccessThreshold: 1, FailureThreshold: 4,
	}
	if pc.Liveness != wantL {
		t.Errorf("liveness mismatch:\n got %+v\nwant %+v", pc.Liveness, wantL)
	}
	if pc.Readiness != wantR {
		t.Errorf("readiness mismatch:\n got %+v\nwant %+v", pc.Readiness, wantR)
	}
	if pc.YAML == "" {
		t.Fatalf("expected non-empty YAML")
	}
	if !strings.Contains(pc.YAML, "fallback defaults") {
		t.Errorf("expected fallback comment in YAML, got:\n%s", pc.YAML)
	}
	for _, frag := range []string{
		"path: /healthz",
		"path: /healthzr",
		"port: 9090",
		"scheme: HTTPS",
		"initialDelaySeconds: 15",
		"initialDelaySeconds: 16",
		"periodSeconds: 20",
		"periodSeconds: 15",
		"timeoutSeconds: 2",
		"timeoutSeconds: 3",
		"failureThreshold: 3",
		"failureThreshold: 4",
	} {
		if !strings.Contains(pc.YAML, frag) {
			t.Errorf("expected YAML to contain %q, got:\n%s", frag, pc.YAML)
		}
	}
}

func TestRenderProbeYAMLFromPod(t *testing.T) {
	pc := defaultProbeConfig()
	pc.Source = "pod"
	pc.PodName = "introspect-abc12"
	pc.Namespace = "demo"
	pc.Container = "introspect"
	// simulate a manifest that omits terminationGracePeriodSeconds and overrides liveness
	pc.Liveness.FailureThreshold = 5
	pc.YAML = renderProbeYAML(pc)

	if !strings.Contains(pc.YAML, "pod introspect-abc12 (namespace demo)") {
		t.Errorf("expected live pod header, got:\n%s", pc.YAML)
	}
	if !strings.Contains(pc.YAML, "- name: introspect") {
		t.Errorf("expected container name, got:\n%s", pc.YAML)
	}
	if !strings.Contains(pc.YAML, "failureThreshold: 5") {
		t.Errorf("expected overridden liveness threshold 5, got:\n%s", pc.YAML)
	}
	if strings.Contains(pc.YAML, "terminationGracePeriodSeconds") {
		t.Errorf("did not expect terminationGracePeriodSeconds when unset, got:\n%s", pc.YAML)
	}
}

func TestRenderProbeYAMLWithTerminationGracePeriod(t *testing.T) {
	pc := defaultProbeConfig()
	pc.Readiness.TerminationGracePeriodSet = true
	pc.Readiness.TerminationGracePeriod = 30
	pc.YAML = renderProbeYAML(pc)
	if !strings.Contains(pc.YAML, "terminationGracePeriodSeconds: 30") {
		t.Errorf("expected terminationGracePeriodSeconds: 30, got:\n%s", pc.YAML)
	}
}
