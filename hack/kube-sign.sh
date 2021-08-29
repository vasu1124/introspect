#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${0})

mkdir -p ${SCRIPT_ROOT}/../etc/tls
cat <<EOF >${SCRIPT_ROOT}/../etc/tls/csr.conf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
[san]
subjectAltName=DNS:introspect,DNS:introspect.default,DNS:introspect.default.svc
EOF

openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -subj "/CN=introspect" \
  -extensions san \
  -keyout ${SCRIPT_ROOT}/../etc/tls/server.key \
  -out    ${SCRIPT_ROOT}/../etc/tls/server.crt \
  -config ${SCRIPT_ROOT}/../etc/tls/csr.conf

openssl x509 -in ${SCRIPT_ROOT}/../etc/tls/server.crt -text -noout

cat <<EOF >${SCRIPT_ROOT}/../kubernetes/introspect/introspect-tls.yaml
apiVersion: v1
kind: Secret
metadata:
  name: introspect-tls
type: Opaque
data:
  server.crt: $(cat ${SCRIPT_ROOT}/../etc/tls/server.crt | base64 | tr -d '\n')
  server.key: $(cat ${SCRIPT_ROOT}/../etc/tls/server.key | base64 | tr -d '\n')
EOF

cat <<EOF >${SCRIPT_ROOT}/../kubernetes/introspect/introspect-validatingwh.yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: introspect-validationwebook
webhooks:
  - name: introspect.default.svc
    namespaceSelector:
      matchLabels:
        validation: enabled
    rules:
      - apiGroups: [""]
        apiVersions: ["v1"]
        operations: ["CREATE"]
        resources: ["pods"]
        scope: "Namespaced"
    clientConfig:
      caBundle: $(cat ${SCRIPT_ROOT}/../etc/tls/server.crt | base64 | tr -d '\n')
      service:
        name: introspect
        namespace: default
        path: "/validate"
        port: 9443
    admissionReviewVersions: ["v1", "v1beta1"]
    sideEffects: None
    timeoutSeconds: 10
    failurePolicy: Ignore
EOF
