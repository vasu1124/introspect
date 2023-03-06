OCIREPO:=ghcr.io/vasu1124
DOCKER_TARGET_PLATFORM:=linux/amd64,linux/arm/v7 #linux/arm64

# nothing to edit beyond this point
BINARY:=introspect
GOARCH:=amd64

gitVersion=$(shell cat introspect.VERSION)
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
		--tag ${OCIREPO}/introspect:${gitVersion} \
		--build-arg gitVersion=${gitVersion} \
		--build-arg gitCommit=${gitCommit} \
		--build-arg gitTreeState=${gitTreeState} \
		--file Dockerfile \
		.
#	docker manifest inspect ${OCIREPO}/introspect:${gitVersion}

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
docker-push: build
	docker push ${OCIREPO}/introspect:${gitVersion}

.PHONY: kubernetes/k8s-visualizer
kubernetes/k8s-visualizer:
#	original was git clone https://github.com/brendandburns/gcp-live-k8s-visualizer.git
	git clone https://github.com/vasu1124/k8s-visualizer.git kubernetes/k8s-visualizer
	echo ./hack/kube-proxy.sh or kubectl proxy --www=./kubernetes/k8s-visualizer/src -p 8001
	echo open browser with http://localhost:8001/static/

ocm/.gen/introspect/introspect-helm-0.1.0.tgz:
	export HELM_EXPERIMENTAL_OCI=1
	mkdir -p ocm/.gen/introspect/
	helm package ./kubernetes/helm/introspect/ --app-version ${gitVersion} -d ocm/.gen/introspect
	helm push ocm/.gen/introspect/introspect-helm-0.1.0.tgz oci://${OCIREPO}/helm

MONGOCHARTVERSION:=12.1.19
MONGOTAG:=4.4.14
ocm/.gen/mongodb/mongodb-${MONGOCHARTVERSION}.tgz: helm-bitnami-repo
	export HELM_EXPERIMENTAL_OCI=1
	mkdir -p ocm/.gen/mongodb/
	helm pull bitnami/mongodb -d ocm/.gen/mongodb --version ${MONGOCHARTVERSION}
	helm push ocm/.gen/mongodb/mongodb-${MONGOCHARTVERSION}.tgz oci://${OCIREPO}/helm

ETCDCHARTVERSION:=8.7.6
ETCDTAG:=3.5.7
ocm/.gen/etcd/etcd-${ETCDCHARTVERSION}.tgz: helm-bitnami-repo
	export HELM_EXPERIMENTAL_OCI=1
	mkdir -p ocm/.gen/etcd/
	helm pull bitnami/etcd -d ocm/.gen/etcd --version ${ETCDCHARTVERSION}
	helm push ocm/.gen/etcd/etcd-${ETCDCHARTVERSION}.tgz oci://${OCIREPO}/helm

.PHONY: helm-bitnami-repo
helm-bitnami-repo:
	helm repo add bitnami https://charts.bitnami.com/bitnami

.PHONY: helm-push
helm-push: ocm/.gen/introspect/introspect-helm-0.1.0.tgz ocm/.gen/mongodb/mongodb-${MONGOCHARTVERSION}.tgz ocm/.gen/etcd/etcd-${ETCDCHARTVERSION}.tgz

.PHONY: ocm
ocm: ./ocm/introspect/component.yaml ./ocm/mongodb/component.yaml ./ocm/etcd/component.yaml ./ocm/app-introspect/component.yaml
	-mkdir -p ocm/.gen
	ocm cv add -cf -F ./ocm/.gen/ctf ./ocm/introspect/component.yaml  \
		OCI=ghcr.io ORG=vasu1124 \
		VERSION=${gitVersion} REF=${gitRefs} COMMIT=${gitCommit} 
	ocm cv add     -F ./ocm/.gen/ctf ./ocm/mongodb/component.yaml        \
		OCI=ghcr.io ORG=vasu1124 \
		VERSION=${MONGOCHARTVERSION} TAG=${MONGOTAG} COMMIT=093d55f1ec11138857ec1b3aa32f7e4d19a32c1d
	ocm cv add     -F ./ocm/.gen/ctf ./ocm/etcd/component.yaml           \
		OCI=ghcr.io ORG=vasu1124 \
		VERSION=${ETCDCHARTVERSION}  TAG=${ETCDTAG}  COMMIT=d8f63d45e8754c0d330e9075f8db22d0b5cdd7ba
	ocm cv add     -F ./ocm/.gen/ctf ./ocm/app-introspect/component.yaml \
		OCI=ghcr.io ORG=vasu1124 \
		VERSION=${gitVersion} MONGODB_VERSION=${MONGOCHARTVERSION} ETCD_VERSION=${ETCDTAG} INTROSPECT_VERSION=${gitVersion} 

.PHONY: ctf-push
ctf-push: ocm
	ocm transfer ctf ./ocm/.gen/ctf ghcr.io/vasu1124/ocm --overwrite



# openssl genpkey -out mysign.key -algorithm RSA
# openssl rsa -in private.key -outform PEM -pubout -out mysign.pub
# component-cli ca signature sign rsa ghcr.io/vasu1124/ocm github.com/vasu1124/app-introspect 1.0.0  --upload-base-url ghcr.io/vasu1124/ocmtest --recursive --signature-name mysign --private-key mysign.key
