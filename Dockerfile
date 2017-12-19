FROM hub.docker.hpecorp.net/protobuf-example/go-build:v0.2.0 as builder
ENV project /go/src/github.hpe.com/monasca/monasca-sidecar

COPY . $project
WORKDIR $project

RUN make depend && make && mv ./bin/monasca-sidecar /

FROM alpine:3.6 as certs

RUN apk --update --no-cache add \
    ca-certificates

COPY --from=builder /monasca-sidecar /

ENTRYPOINT ["/monasca-sidecar"]

ARG TAG
ARG GIT_SHA
ARG BUILD_DATE

ENV TAG $TAG
ENV GIT_SHA $GIT_SHA
ENV BUILD_DATE $BUILD_DATE
