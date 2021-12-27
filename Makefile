DOCKERREPO:=ghcr.io/vasu1124
DOCKER_TARGET_PLATFORM:=linux/amd64,linux/arm/v7 #linux/arm64

# nothing to edit beyond this point
BINARY:=introspect
GOARCH:=amd64

VERSION=$(shell cat introspect.VERSION)
COMMIT:=$(shell git rev-parse HEAD)
BRANCH:=$(shell git rev-parse --abbrev-ref HEAD)

LDFLAGS=-ldflags "-X github.com/vasu1124/introspect/pkg/version.Version=${VERSION} -X github.com/vasu1124/introspect/pkg/version.Commit=${COMMIT} -X github.com/vasu1124/introspect/pkg/version.Branch=${BRANCH}"

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
	-rm -f ${BINARY}-* debug kubernetes/k14s/kbld.lock.yaml

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
		--tag ${DOCKERREPO}/introspect:${VERSION} \
		--build-arg VERSION=${VERSION} \
		--build-arg COMMIT=${COMMIT} \
		--build-arg BRANCH=${BRANCH} \
		--file Dockerfile \
		.
	docker manifest inspect ${DOCKERREPO}/introspect:${VERSION}

.PHONY: buildx
buildx: ${SOURCES}
	docker buildx build \
		--output "type=image,push=false" \
		--platform ${DOCKER_TARGET_PLATFORM} \
		--tag ${DOCKERREPO}/introspect:${VERSION} \
		--build-arg VERSION=${VERSION} \
		--build-arg COMMIT=${COMMIT} \
		--build-arg BRANCH=${BRANCH} \
		--file Dockerfile \
		.
	docker buildx imagetools inspect ${DOCKERREPO}/introspect:${VERSION}

.PHONY: deploy
deploy:
	kubernetes/k14s/kapp-deploy.sh

.PHONY: docker
docker: docker/scratch.docker docker/alpine.docker docker/ubuntu.docker

docker/scratch.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.scratch
	docker build -f docker/Dockerfile.scratch \
		--tag ${DOCKERREPO}/introspect-scratch:${VERSION} \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
		.
	touch docker/scratch.docker

docker/alpine.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.alpine
	docker build -f docker/Dockerfile.alpine \
		--tag ${DOCKERREPO}/introspect:${VERSION} \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
	 	.
	touch docker/alpine.docker

docker/ubuntu.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.ubuntu
	docker build -f docker/Dockerfile.ubuntu \
		--tag ${DOCKERREPO}/introspect-ubuntu:${VERSION} \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
	 	.
	touch docker/ubuntu.docker

.PHONY: 1.0.0
1.0.0:
	echo "1.0.0" >introspect.VERSION
	VERSION="1.0.0"

.PHONY: 2.0.0
2.0.0:
	echo "2.0.0" >introspect.VERSION
	VERSION="2.0.0"

# we are only pushing alpine
.PHONY: docker-push
docker-push: docker/alpine.docker
	docker push ${DOCKERREPO}/introspect:${VERSION}

kubernetes/k8s-visualizer:
#	original was git clone https://github.com/brendandburns/gcp-live-k8s-visualizer.git
	git clone https://github.com/vasu1124/k8s-visualizer.git kubernetes/k8s-visualizer
	echo ./hack/kube-proxy.sh or kubectl proxy --www=./kubernetes/k8s-visualizer/src -p 8001
	echo open browser with http://localhost:8001/static/
