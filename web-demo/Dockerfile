# syntax=docker/dockerfile:1.3

FROM alpine:latest
RUN apk update && apk add --no-cache \
  make \
  libpcap-dev \
  bash \
  musl-dev \
  g++ \
  git \
  tar \
  gzip \
  curl \
  go \
  openssh
ENV GOROOT /usr/lib/go
ENV GOPATH /go
ENV PATH /go/bin:$PATH

ARG GIT_COMMIT_HASH
# XXX <05-02-22, afjoseph> ARG variables won't shell expand with ENTRYPOINT and
# CMD directives: only ENV can do that

COPY ./spa-server /spa-server
COPY ./frontend/build /frontend-build
WORKDIR /spa-server
# XXX <05-02-22, afjoseph> Using -ldflags inline doesn't work:
# https://github.com/google/ko/issues/167
ENV GOFLAGS "-ldflags=-X=main.gitCommitHash=$GIT_COMMIT_HASH"
ENTRYPOINT go run . -spa-build-path=../frontend-build -port=8080
