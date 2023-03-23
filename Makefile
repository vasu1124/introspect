# SPDX-FileCopyrightText: 2018 vasu1124
#
# SPDX-License-Identifier: Apache-2.0

include .env
export

DOCKER_TARGET_PLATFORM:=linux/amd64 #,linux/arm/v7 #linux/arm64

BINARY:=introspect
GOARCH:=amd64

gitVersion:=${INTROSPECT_VERSION}
gitCommit:=$(shell git rev-parse --verify HEAD)
gitRefs:=$(shell git symbolic-ref HEAD)
gitTreeState=$(shell [ -z git status --porcelain 2>/dev/null ] && echo clean || echo dirty)
buildDate:=$(shell date --rfc-3339=seconds | sed 's/ /T/')

LDFLAGS=-ldflags \
	"-X github.com/vasu1124/introspect/pkg/version.gitVersion=${gitVersion} \
 	 -X github.com/vasu1124/introspect/pkg/version.gitCommit=${gitCommit} \
	 -X github.com/vasu1124/introspect/pkg/version.gitTreeState=${gitTreeState} \
	 -X github.com/vasu1124/introspect/pkg/version.buildDate=${buildDate}"

# Build the project
ifeq ($(shell uname -s), Darwin)
    all=${BINARY}-darwin-${GOARCH} ${BINARY}-linux-${GOARCH} docker/alpine.docker
else
    all=${BINARY}-linux-${GOARCH} docker/alpine.docker
endif
all: ${all}

tlsfiles := kubernetes/introspect/introspect-validatingwh.yaml kubernetes/introspect/introspect-tls.yaml etc/tls/csr.conf etc/tls/server.crt etc/tls/server.key 
.PHONY: tls
tls: ${tlsfiles}
	hack/kube-sign.sh

.PHONY: clean
clean:
	-rm -rf ${BINARY}-* debug kubernetes/k14s/kbld.lock.yaml ocm/.gen

# kubebuilder: Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests:
	cd pkg/operator/useless; make manifests

# Generate code
.PHONY: generate
generate:
	cd pkg/operator/useless; make generate

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Run tests
.PHONY: test
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out


${GOPATH}/bin/cfssl:
	go env
	-mkdir ${GOPATH}/bin
	go get -u github.com/cloudflare/cfssl/cmd/cfssl
	go get -u github.com/cloudflare/cfssl/cmd/cfssljson


# SOURCES := $(shell find . -type f -name '*.go')
SOURCES := $(shell go list -f '{{$$I:=.Dir}}{{range .GoFiles }}{{$$I}}/{{.}} {{end}}' ./... )

${BINARY}-linux-${GOARCH}: ${SOURCES}
	CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-linux-${GOARCH} ./cmd
	rm -f kubernetes/k14s/kbld.lock.yaml

${BINARY}-darwin-${GOARCH}: ${SOURCES}
	CGO_ENABLED=0 GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-darwin-${GOARCH} ./cmd
	rm -f kubernetes/k14s/kbld.lock.yaml

.PHONY: build
build: ${SOURCES}
	docker build \
		--tag ${OCI}/${ORG}/introspect:${gitVersion} \
		--build-arg gitVersion=${gitVersion} \
		--build-arg gitCommit=${gitCommit} \
		--build-arg gitTreeState=${gitTreeState} \
		--file Dockerfile \
		.
	docker buildx imagetools inspect ${OCI}/${ORG}/introspect:${gitVersion}
#	docker manifest inspect ${OCI}/${ORG}/introspect:${gitVersion}

.PHONY: buildx
buildx: ${SOURCES}
	docker buildx build \
		--output "type=image,push=false" \
		--platform ${DOCKER_TARGET_PLATFORM} \
		--tag ${OCI}/${ORG}/introspect:${gitVersion} \
		--build-arg gitVersion=${gitVersion} \
		--build-arg gitCommit=${gitCommit} \
		--build-arg gitTreeState=${gitTreeState} \
		--file Dockerfile \
		.
	docker buildx imagetools inspect ${OCI}/${ORG}/introspect:${gitVersion}

.PHONY: deploy
deploy:
	kubernetes/k14s/kapp-deploy.sh

.PHONY: docker
docker: docker/scratch.docker docker/alpine.docker docker/ubuntu.docker

docker/scratch.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.scratch
	docker build -f docker/Dockerfile.scratch \
		--tag ${OCI}/${ORG}/introspect-scratch:${gitVersion} \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
		.
	touch docker/scratch.docker

docker/alpine.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.alpine
	docker build -f docker/Dockerfile.alpine \
		--tag ${OCI}/${ORG}/introspect:${gitVersion} \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
	 	.
	touch docker/alpine.docker

docker/ubuntu.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.ubuntu
	docker build -f docker/Dockerfile.ubuntu \
		--tag ${OCI}/${ORG}/introspect-ubuntu:${gitVersion} \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
	 	.
	touch docker/ubuntu.docker

.PHONY: docker-push
docker-push:
	docker push ${OCI}/${ORG}/introspect:${gitVersion}

.PHONY: kubernetes/k8s-visualizer
kubernetes/k8s-visualizer:
#	original was git clone https://github.com/brendandburns/gcp-live-k8s-visualizer.git
	git clone https://github.com/vasu1124/k8s-visualizer.git kubernetes/k8s-visualizer
	echo ./hack/kube-proxy.sh or kubectl proxy --www=./kubernetes/k8s-visualizer/src -p 8001
	echo open browser with http://localhost:8001/static/

ocm/.gen/introspect/introspect-helm-${INTROSPECT_VERSION}.tgz:
	mkdir -p ocm/.gen/introspect/
	helm package ./kubernetes/helm/introspect/ --app-version ${INTROSPECT_VERSION} -d ocm/.gen/introspect
#	helm push ocm/.gen/introspect/introspect-helm-${INTROSPECT_VERSION}.tgz oci://${OCI}/${ORG}/helm

ocm/.gen/mongodb/mongodb-${MONGODB_CHART}.tgz:
	mkdir -p ocm/.gen/mongodb/
	helm pull mongodb -d ocm/.gen/mongodb --version ${MONGODB_CHART} --repo https://charts.bitnami.com/bitnami
#	helm push ocm/.gen/mongodb/mongodb-${MONGOCHARTVERSION}.tgz oci://${OCI}/${ORG}/helm

ocm/.gen/etcd/etcd-${ETCD_CHART}.tgz:
	mkdir -p ocm/.gen/etcd/
	helm pull etcd -d ocm/.gen/etcd --version ${ETCD_CHART} --repo https://charts.bitnami.com/bitnami
#	helm push ocm/.gen/etcd/etcd-${ETCD_CHART}.tgz oci://${OCI}/${ORG}/helm

.PHONY: helm
helm: ocm/.gen/introspect/introspect-helm-${INTROSPECT_VERSION}.tgz ocm/.gen/mongodb/mongodb-${MONGODB_CHART}.tgz ocm/.gen/etcd/etcd-${ETCD_CHART}.tgz

.PHONY: ./ocm/.gen/dynamic.yaml
.ONESHELL:
./ocm/.gen/dynamic.yaml:
	-mkdir -p ocm/.gen
	cat <<- EOF >$@
		$$(cat .env | tr "=" ":")
		INTROSPECT_REF: ${gitRefs}
		INTROSPECT_COMMIT: ${gitCommit} 
	EOF

.PHONY: ocm
ocm: helm ./ocm/.gen/dynamic.yaml ./ocm/introspect/component.yaml ./ocm/mongodb/component.yaml ./ocm/etcd/component.yaml ./ocm/app-introspect/component.yaml
	ocm cv add -cf -F ./ocm/.gen/ctf ./ocm/introspect/component.yaml  \
		--settings ./ocm/introspect/settings.yaml \
		--settings ./ocm/.gen/dynamic.yaml
	ocm cv add     -F ./ocm/.gen/ctf ./ocm/mongodb/component.yaml     \
		--settings ./ocm/mongodb/settings.yaml \
		--settings ./ocm/.gen/dynamic.yaml 
	ocm cv add     -F ./ocm/.gen/ctf ./ocm/etcd/component.yaml        \
		--settings ./ocm/etcd/settings.yaml \
		--settings ./ocm/.gen/dynamic.yaml 
	ocm cv add     -F ./ocm/.gen/ctf ./ocm/app-introspect/component.yaml \
		--settings ./ocm/app-introspect/settings.yaml \
		--settings ./ocm/.gen/dynamic.yaml 

.PHONY: ctf-push
ctf-push: ocm
	ocm transfer ctf ./ocm/.gen/ctf ${OCI}/${ORG}/ocm --overwrite

# openssl genpkey -out mysign.key -algorithm RSA
# openssl rsa -in private.key -outform PEM -pubout -out mysign.pub
# component-cli ca signature sign rsa ghcr.io/vasu1124/ocm github.com/vasu1124/app-introspect 1.0.0  --upload-base-url ghcr.io/vasu1124/ocmtest --recursive --signature-name mysign --private-key mysign.key
