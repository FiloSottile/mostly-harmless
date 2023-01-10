FROM golang:1.20rc2-alpine3.16 AS builder

RUN apk add --no-cache build-base

COPY *.go go.mod go.sum src
WORKDIR src
RUN go install -trimpath

FROM alpine:3.16

COPY --from=builder /go/bin/bastion /usr/local/bin/
COPY example_backends.txt /etc/bastion/backends.txt
VOLUME /usr/local/lib/autocert

ENTRYPOINT ["bastion", "-listen", ":8443", \
    "-backends", "/etc/bastion/backends.txt", \
    "-cache", "/usr/local/lib/autocert", \
    "-host", "example-bastion.fly.dev", \
    "-email", "example-bastion@bip.filippo.io"]
