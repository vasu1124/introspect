# SPDX-FileCopyrightText: 2018 vasu1124
#
# SPDX-License-Identifier: GPL-3.0-or-later

FROM golang:1.24-alpine as builder
ARG gitVersion=0.0.0-dev
ARG gitCommit=0000000000000000000000000000000000000000
ARG gitTreeState="dirty"

WORKDIR /introspect
# RUN GO111MODULE=off go get github.com/go-delve/delve/cmd/dlv
COPY go.* ./
COPY cmd cmd
COPY pkg pkg
COPY .env ./ 
RUN buildDate=$(date -I'seconds'); \
    go build \
    -ldflags "\
	-X github.com/vasu1124/introspect/pkg/version.gitVersion=${gitVersion} \
 	-X github.com/vasu1124/introspect/pkg/version.gitCommit=${gitCommit} \
	-X github.com/vasu1124/introspect/pkg/version.gitTreeState=${gitTreeState} \
	-X github.com/vasu1124/introspect/pkg/version.buildDate=${buildDate} \
    " \
    -o introspect-linux ./cmd

# final stage
FROM alpine:3
LABEL maintainer="vasu1124@actvirtual.com" \
    immutable.labels=true \
    org.opencontainers.image.vendor="actvirtual" \
    org.opencontainers.image.licenses="GPL-3.0" \
    org.opencontainers.image.title="Introspect" \
    org.opencontainers.image.source="https://github.com/vasu1124/introspect" \
    org.opencontainers.image.description="DemoSuite for Kubernetes" \
    com.actvirtual.quality="evaluation" \
    com.actvirtual.product="DemoSuite"

WORKDIR /introspect
RUN apk --no-cache add --update bash ca-certificates libc6-compat \
    && rm -rf /var/cache/apk/*
# COPY --from=builder /go/bin/dlv ./
COPY --from=builder /introspect/introspect-linux ./
COPY tmpl tmpl
COPY css css

EXPOSE 9090 9443
CMD ["./introspect-linux"]

# If you want to use the debugger, you need to modify  the
# container and point it to the "dlv debug" command:
# Start the "dlv debug" server on port 3000 of the container. (Note that the
# application process will NOT start until the debugger is attached.)
# EXPOSE 3000
# CMD ["./dlv", "debug", "./cmd",  "--api-version=2", "--headless", "--listen=:3000", "--log"]
