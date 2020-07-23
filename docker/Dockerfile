ARG UPSTREAM_TAG=1.20.0

FROM golang:1.14.6-alpine3.12 as builder
ARG GO_CODECOMMIT_VER
ARG GO_CODECOMMIT_COMMIT

ENV GOPATH /go
ENV USER root

COPY . /tmp/build

RUN set -ex && \
     cd /tmp/build/cmd/codecommit && \
     GO111MODULE=on go build -mod vendor -ldflags "-w -s -X main.Version=$GO_CODECOMMIT_VER -X main.GitCommit=$GO_CODECOMMIT_COMMIT" && \
     find . && \
     echo "Build of go-codecommit $GO_CODECOMMIT_VER complete."

FROM alpine:3.12.0
COPY --from=builder /tmp/build/cmd/codecommit/codecommit  /usr/local/bin/.
RUN set -ex && \
     apk update --no-cache && \
     apk add --no-cache git
