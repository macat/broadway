# registry.namely.tech/namely/broadway-dev:v4
# If you change this file, or any of the dependencies, build a new image and
# increase the version number.
FROM golang:1.6.0-alpine
RUN apk add --update git && rm -rf /var/cache/apk/*

RUN go get github.com/tools/godep github.com/kisielk/errcheck github.com/golang/lint/golint

RUN mkdir -p /go/src/github.com/namely/broadway
WORKDIR /go/src/github.com/namely/broadway
