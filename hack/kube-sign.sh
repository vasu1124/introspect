#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${0})

kubectl delete csr introspect.default || true

cat <<EOF | kubectl create -f -
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: introspect.default
spec:
  groups:
  - system:authenticated
  request: $(cat etc/mycerts/webhook.csr | base64 | tr -d '\n')
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF

kubectl certificate approve introspect.default
kubectl get csr
kubectl describe csr introspect.default

kubectl get csr introspect.default -o jsonpath='{.status.certificate}' \
  > ${SCRIPT_ROOT}/../etc/mycerts/webhook.b64

kubectl get csr introspect.default -o jsonpath='{.status.certificate}' \
 | base64 --decode > ${SCRIPT_ROOT}/../etc/mycerts/webhook.pem

