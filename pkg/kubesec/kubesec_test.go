package kubesec

import (
	"testing"

	yaml "gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/* Eample Response
{
  "score": -29,
  "scoring": {
    "critical": [
      {
        "selector": "containers[] .securityContext .privileged == true",
        "reason": "Privileged containers can allow almost completely unrestricted host access"
      }
    ],
    "advise": [
      {
        "selector": "containers[] .securityContext .runAsNonRoot == true",
        "reason": "Force the running image to run as a non-root user to ensure least privilege"
      },
      {
        "selector": "containers[] .securityContext .capabilities .drop",
        "reason": "Reducing kernel capabilities available to a container limits its attack surface",
        "href": "https://kubernetes.io/docs/tasks/configure-pod-container/security-context/"
      },
      {
        "selector": "containers[] .securityContext .runAsUser > 10000",
        "reason": "Run as a high-UID user to avoid conflicts with the host's user table"
      },
      {
        "selector": "containers[] .securityContext .capabilities .drop | index(\"ALL\")",
        "reason": "Drop all capabilities and add only those required to reduce syscall attack surface"
      },
      {
        "selector": "containers[] .resources .limits .memory",
        "reason": "Enforcing memory limits prevents DOS via resource exhaustion"
      }
    ]
  }
}
*/

func TestGetKubesecYAML(t *testing.T) {
	var yamlStr = []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: kubesec-demo
spec:
  containers:
  - name: kubesec-demo
    image: gcr.io/google-samples/node-hello:1.0
    securityContext:
      privileged: true
      readOnlyRootFilesystem: true
  `)

	t.Log(string(yamlStr))

	k, _ := GetKubesecPodYAML(yamlStr)
	t.Logf("%v\n", k)
	if k == nil || k.Error != "" {
		t.Fail()
	} else {
		t.Logf("Score %d\n", k.Score)
	}
}

func TestGetKubesecPod(t *testing.T) {

	truep := true
	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo",
			Labels: map[string]string{
				"app": "demo",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "web",
					Image: "gcr.io/google-samples/node-hello:1.0",
					Ports: []v1.ContainerPort{
						{
							Name:          "http",
							Protocol:      v1.ProtocolTCP,
							ContainerPort: 80,
						},
					},
					SecurityContext: &v1.SecurityContext{
						Privileged:             &truep,
						ReadOnlyRootFilesystem: &truep,
					},
				},
			},
		},
	}

	yamlStr, _ := yaml.Marshal(pod)
	t.Log(string(yamlStr))

	k, _ := GetKubesecPod(&pod)
	t.Logf("%v\n", k)
	if k == nil || k.Error != "" {
		t.Fail()
	} else {
		t.Logf("Score %d\n", k.Score)
	}
}
