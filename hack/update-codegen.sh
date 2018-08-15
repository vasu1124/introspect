#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${0})
CODEGEN_PKG=${SCRIPT_ROOT}/../vendor/k8s.io/code-generator

${CODEGEN_PKG}/generate-groups.sh \
  "deepcopy" \
  github.com/vasu1124/introspect/pkg/operator \
  github.com/vasu1124/introspect/pkg/operator/apis \
  useless:v1 
