# (C) Copyright 2017 Hewlett Packard Enterprise Development LP

FROM alpine:3.6 as go-builder

ARG SIDECAR_REPO=https://github.hpe.com/monasca/monasca-sidecar
ARG SIDECAR_BRANCH=master

ENV GOPATH=/go CGO_ENABLED=0 GOOS=linux

# To force a rebuild, pass --build-arg REBUILD="$(DATE)" when running
# `docker build`
ARG REBUILD=1

RUN apk add --no-cache git go glide make g++ openssl-dev musl-dev
RUN mkdir -p $GOPATH/src/github.hpe.com/monasca/monasca-sidecar

WORKDIR $GOPATH/src/github.hpe.com/monasca/monasca-sidecar

RUN git init && \
    git remote add origin $SIDECAR_REPO && \
    git fetch origin $SIDECAR_BRANCH && \
    git reset --hard FETCH_HEAD

RUN glide install && \
    go build -a -o ./sidecar

FROM alpine:3.6

RUN apk add --no-cache ca-certificates tini

COPY --from=go-builder \
    /go/src/github.hpe.com/monasca/monasca-sidecar/sidecar \
    /sidecar

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/sidecar"]ILD_DATE $BUILD_DATE
