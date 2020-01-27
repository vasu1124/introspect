FROM golang:1.13-stretch as builder
ARG VERSION_TO_BUILD
ENV VERSION_TO_BUILD=$VERSION_TO_BUILD
LABEL maintainer="vasu1124@actvirtual.com" \
      immutable.labels=true \
      org.opencontainers.image.vendor="actvirtual" \
      org.opencontainers.image.licenses="GPL-3.0" \
      org.opencontainers.image.title="Introspect" \
      org.opencontainers.image.source="https://github.com/vasu1124/introspect" \
      com.actvirtual.quality="evaluation" \
      com.actvirtual.product="DemoSuite" 

WORKDIR /go/src/github.com/vasu1124/introspect
COPY go.mod .
RUN go get -u github.com/go-delve/delve/cmd/dlv

COPY tmpl tmpl
COPY css css
COPY cmd cmd
COPY pkg pkg
#COPY vendor vendor
COPY introspect.VERSION .
COPY .git .
COPY Makefile .

#RUN make depensure clean
RUN make $VERSION_TO_BUILD introspect-linux-amd64 


 
#RUN apk --no-cache add --update bash ca-certificates libc6-compat \
#    && rm -rf /var/cache/apk/*

EXPOSE 9090

# If you want to use the debugger, you need to modify  the
# container and point it to the "dlv debug" command:
# Start the "dlv debug" server on port 3000 of the container. (Note that the
# application process will NOT start until the debugger is attached.)
EXPOSE 3000
CMD ["/go/bin/dlv", "debug", "./cmd/introspect",  "--api-version=2", "--headless", "--listen=:3000", "--log"]

#CMD ["./introspect-linux-amd64"]