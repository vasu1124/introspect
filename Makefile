OCIREPO:=ghcr.io/vasu1124
DOCKER_TARGET_PLATFORM:=linux/amd64,linux/arm/v7 #linux/arm64

# nothing to edit beyond this point
BINARY:=introspect
GOARCH:=amd64

gitVersion=$(shell cat introspect.VERSION)
gitCommit:=$(shell git rev-parse --verify HEAD)
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
		--tag ${OCIREPO}/introspect:${gitVersion} \
		--build-arg gitVersion=${gitVersion} \
		--build-arg gitCommit=${gitCommit} \
		--build-arg gitTreeState=${gitTreeState} \
		--file Dockerfile \
		.
	docker manifest inspect ${OCIREPO}/introspect:${gitVersion}

.PHONY: buildx
buildx: ${SOURCES}
	docker buildx build \
		--output "type=image,push=false" \
		--platform ${DOCKER_TARGET_PLATFORM} \
		--tag ${OCIREPO}/introspect:${gitVersion} \
		--build-arg gitVersion=${gitVersion} \
		--build-arg gitCommit=${gitCommit} \
		--build-arg gitTreeState=${gitTreeState} \
		--file Dockerfile \
		.
	docker buildx imagetools inspect ${OCIREPO}/introspect:${gitVersion}

.PHONY: deploy
deploy:
	kubernetes/k14s/kapp-deploy.sh

.PHONY: docker
docker: docker/scratch.docker docker/alpine.docker docker/ubuntu.docker

docker/scratch.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.scratch
	docker build -f docker/Dockerfile.scratch \
		--tag ${OCIREPO}/introspect-scratch:${gitVersion} \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
		.
	touch docker/scratch.docker

docker/alpine.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.alpine
	docker build -f docker/Dockerfile.alpine \
		--tag ${OCIREPO}/introspect:${gitVersion} \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
	 	.
	touch docker/alpine.docker

docker/ubuntu.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.ubuntu
	docker build -f docker/Dockerfile.ubuntu \
		--tag ${OCIREPO}/introspect-ubuntu:${gitVersion} \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
	 	.
	touch docker/ubuntu.docker

# we are only pushing alpine
.PHONY: docker-push
docker-push: docker/alpine.docker
	docker push ${OCIREPO}/introspect:${gitVersion}

.PHONY: kubernetes/k8s-visualizer
kubernetes/k8s-visualizer:
#	original was git clone https://github.com/brendandburns/gcp-live-k8s-visualizer.git
	git clone https://github.com/vasu1124/k8s-visualizer.git kubernetes/k8s-visualizer
	echo ./hack/kube-proxy.sh or kubectl proxy --www=./kubernetes/k8s-visualizer/src -p 8001
	echo open browser with http://localhost:8001/static/

.PHONY: cd
cd:
	component-cli component-archive create --component-name github.com/vasu1124/introspect  --component-version ${gitVersion} ./ocm/.gen/component
	component-cli component-archive resource add  ./ocm/.gen/component OCI=ghcr.io ORG=vasu1124 gitVersion=${gitVersion} ./ocm/resources.yaml
	component-cli component-archive sources  add  ./ocm/.gen/component OCI=ghcr.io ORG=vasu1124 gitVersion=${gitVersion} ./ocm/sources.yaml

.PHONY: ctf
ctf: cd
	component-cli ctf add ./ocm/.gen/ctf -f ./ocm/.gen/component

.PHONY: ctfpush
ctfpush: ctf
	component-cli ctf push ./ocm/.gen/ctf --repo-ctx ghcr.io/vasu1124/ocm
