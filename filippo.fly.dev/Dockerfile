FROM golang:1.22.1-alpine3.19 AS builder

COPY . src
WORKDIR src
ENV CGO_ENABLED 0
RUN go install -trimpath

FROM alpine:3.19.1

COPY --from=builder /go/bin/filippo.fly.dev /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/filippo.fly.dev"]
