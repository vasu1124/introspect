DOCKERREPO:=vasu1124

# nothing to edit beyond this point
BINARY:=introspect
GOARCH:=amd64

VERSION=$(shell cat introspect.VERSION)
COMMIT:=$(shell git rev-parse HEAD)
BRANCH:=$(shell git rev-parse --abbrev-ref HEAD)

LDFLAGS=-ldflags "-X github.com/vasu1124/introspect/version.Version=${VERSION} -X github.com/vasu1124/introspect/version.Commit=${COMMIT} -X github.com/vasu1124/introspect/version.Branch=${BRANCH}"

# Build the project
ifeq ($(shell uname -s), Darwin)
    all=${BINARY}-darwin-${GOARCH} ${BINARY}-linux-${GOARCH} docker
else
    all=${BINARY}-linux-${GOARCH} docker
endif
all: ${all}

clean:
	-rm -f ${BINARY}-* debug

dep:
	-mkdir ${GOPATH}/bin
	go get -v -u github.com/golang/dep/cmd/dep
	${GOPATH}/bin/dep ensure -v
	./hack/update-codegen.sh

SOURCES := $(shell find . -type f -name '*.go')

${BINARY}-linux-${GOARCH}: ${SOURCES}
	CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-linux-${GOARCH} . 

${BINARY}-linux-${GOARCH}-1.9: ${SOURCES} 
	docker run --rm -v ${GOPATH}:/go -w /go/src/actvirtual.com/inspect golang:1.9 go build ${LDFLAGS} -o ${BINARY}-linux-${GOARCH} . 

${BINARY}-darwin-${GOARCH}: ${SOURCES}
	CGO_ENABLED=0 GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-darwin-${GOARCH} . 

docker: docker/scratch.docker docker/alpine.docker docker/ubuntu.docker docker/opensuse.docker

docker/scratch.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.scratch
	docker build -f docker/Dockerfile.scratch \
		-t="${DOCKERREPO}/goscratch:${VERSION}" \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
		.
	touch docker/scratch.docker
# docker run --rm -p 8081:8080 ${DOCKERREPO}/goscratch:v1.0

docker/alpine.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.alpine
	docker build -f docker/Dockerfile.alpine \
		-t="${DOCKERREPO}/goalpine:${VERSION}" \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
	 	.
	touch docker/alpine.docker
# docker run --rm -p 8081:8080 ${DOCKERREPO}/goalpine:v1.0

docker/ubuntu.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.ubuntu
	docker build -f docker/Dockerfile.ubuntu \
		-t="${DOCKERREPO}/goubuntu:${VERSION}" \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
	 	.
	touch docker/ubuntu.docker
# docker run --rm -p 8081:8080 ${DOCKERREPO}/goubuntu:v1.0

docker/opensuse.docker: ${BINARY}-linux-${GOARCH} docker/Dockerfile.opensuse
	docker build -f docker/Dockerfile.opensuse \
		-t="${DOCKERREPO}/goopensuse:${VERSION}" \
		--build-arg http_proxy=${http_proxy} \
		--build-arg https_proxy=${https_proxy} \
		--build-arg no_proxy=${no_proxy} \
	 	.
	touch docker/opensuse.docker
# docker run --rm -p 8081:8080 ${DOCKERREPO}/goopensuse:v1.0

v1.0:
	echo "v1.0" >introspect.VERSION
	VERSION="v1.0"

v2.0:
	echo "v2.0" >introspect.VERSION
	VERSION="v2.0"

# we are only pushing alpine
docker-push: docker/alpine.docker
	docker push ${DOCKERREPO}/goalpine:${VERSION}

kubernetes/k8s-visualizer:
#	original was git clone https://github.com/brendandburns/gcp-live-k8s-visualizer.git
	git clone https://github.com/vasu1124/k8s-visualizer.git kubernetes/k8s-visualizer
	echo ./hack/kube-proxy.sh or kubectl proxy --www=./kubernetes/k8s-visualizer/src -p 8001
	echo open browset with http://localhost:8001/static/