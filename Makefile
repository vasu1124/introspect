DOCKERREPO:=vasu1124
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

.PHONY: clean
clean:
	-rm -f ${BINARY}-* debug go.sum ${TLSintermidiate} kubernetes/ValidatingWebhookConfiguration.yaml kubernetes/k14s/kbld.lock.yaml

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all

# Run go fmt against code
.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
.PHONY: vet
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
.PHONY: generate
generate:
	go generate ./pkg/... ./cmd/...

# Run tests
.PHONY: test
test: generate fmt vet manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

.PHONY: deepcopy
deepcopy:
	./hack/update-codegen.sh

${GOPATH}/bin/cfssl:
	go env
	-mkdir ${GOPATH}/bin
	go get -u github.com/cloudflare/cfssl/cmd/cfssl
	go get -u github.com/cloudflare/cfssl/cmd/cfssljson

TLSintermidiate :=  etc/mycerts/webhook.csr etc/mycerts/webhook-key.pem etc/mycerts/webhook.pem etc/mycerts/webhook.b64
TLS: ${GOPATH}/bin/cfssl ${TLSintermidiate} kubernetes/ValidatingWebhookConfiguration.yaml
${TLSintermidiate}: etc/mycerts/webhook.json
	cfssl genkey etc/mycerts/webhook.json | cfssljson -bare etc/mycerts/webhook
	hack/kube-sign.sh

kubernetes/ValidatingWebhookConfiguration.yaml:
	sed -e "s/\$${caBundle}/$$(cat etc/mycerts/webhook.b64)/" <$@.template >$@

# SOURCES := $(shell find . -type f -name '*.go')
SOURCES := $(shell go list -f '{{$$I:=.Dir}}{{range .GoFiles }}{{$$I}}/{{.}} {{end}}' ./... )

${BINARY}-linux-${GOARCH}: ${SOURCES}
	CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-linux-${GOARCH} ./cmd/introspect
	rm -f kubernetes/k14s/kbld.lock.yaml

${BINARY}-darwin-${GOARCH}: ${SOURCES}
	CGO_ENABLED=0 GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-darwin-${GOARCH} ./cmd/introspect
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
docker: docker/scratch.docker docker/alpine.docker docker/ubuntu.docker docker/opensuse.docker

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

docker/opensuse.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.opensuse
	docker build -f docker/Dockerfile.opensuse \
		--tag ${DOCKERREPO}/introspect-opensuse:${VERSION} \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
	 	.
	touch docker/opensuse.docker

.PHONY: v1.0
v1.0:
	echo "v1.0" >introspect.VERSION
	VERSION="v1.0"

.PHONY: v2.0
v2.0:
	echo "v2.0" >introspect.VERSION
	VERSION="v2.0"

# we are only pushing alpine
.PHONY: docker-push
docker-push: docker/alpine.docker
	docker push ${DOCKERREPO}/introspect:${VERSION}

kubernetes/k8s-visualizer:
#	original was git clone https://github.com/brendandburns/gcp-live-k8s-visualizer.git
	git clone https://github.com/vasu1124/k8s-visualizer.git kubernetes/k8s-visualizer
	echo ./hack/kube-proxy.sh or kubectl proxy --www=./kubernetes/k8s-visualizer/src -p 8001
	echo open browser with http://localhost:8001/static/
