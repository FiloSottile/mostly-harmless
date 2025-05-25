FROM golang:1.24.3-alpine3.21 AS builder

COPY . src
WORKDIR src
ENV CGO_ENABLED 0
RUN go install -trimpath

FROM alpine:3.21

COPY --from=builder /go/bin/filippo.fly.dev /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/filippo.fly.dev"]
